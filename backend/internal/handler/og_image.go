package handler

import (
	"fmt"
	"html"
	"math"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	mw "github.com/thask/backend/internal/middleware"
	"github.com/thask/backend/internal/model"
)

var typeColors = map[model.NodeType]string{
	model.NodeTypeFlow:   "#e2b340",
	model.NodeTypeBranch: "#a78bfa",
	model.NodeTypeTask:   "#7ca3c4",
	model.NodeTypeBug:    "#e05252",
	model.NodeTypeAPI:    "#5ea87a",
	model.NodeTypeUI:     "#d4915a",
	model.NodeTypeGroup:  "#7c7570",
}

// OGImage renders an SVG preview of the project graph.
func (h *NodeHandler) OGImage(c echo.Context) error {
	ctx := c.Request().Context()
	projectID := mw.ResolveProjectID(c)

	nodes, err := h.nodeRepo.FindByProjectID(ctx, projectID, nil, nil)
	if err != nil {
		return c.String(http.StatusInternalServerError, "")
	}

	edges, err := h.edgeRepo.FindByProjectID(ctx, projectID)
	if err != nil {
		return c.String(http.StatusInternalServerError, "")
	}

	const imgW, imgH = 1200.0, 630.0

	if len(nodes) == 0 {
		svg := renderEmptySVG("Empty Graph", imgW, imgH)
		c.Response().Header().Set("Content-Type", "image/svg+xml")
		c.Response().Header().Set("Cache-Control", "public, max-age=300")
		return c.String(http.StatusOK, svg)
	}

	// Build position map
	posMap := make(map[string]model.Node)
	for _, n := range nodes {
		posMap[n.ID] = n
	}

	// Calculate bounds
	const pad = 60.0
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, n := range nodes {
		nw, nh := nodeSize(n)
		if n.PositionX-nw/2 < minX {
			minX = n.PositionX - nw/2
		}
		if n.PositionY-nh/2 < minY {
			minY = n.PositionY - nh/2
		}
		if n.PositionX+nw/2 > maxX {
			maxX = n.PositionX + nw/2
		}
		if n.PositionY+nh/2 > maxY {
			maxY = n.PositionY + nh/2
		}
	}

	graphW := maxX - minX
	graphH := maxY - minY
	if graphW < 1 {
		graphW = 1
	}
	if graphH < 1 {
		graphH = 1
	}

	scale := math.Min((imgW-pad*2)/graphW, (imgH-pad*2-40)/graphH)
	if scale > 2 {
		scale = 2
	}
	offsetX := (imgW - graphW*scale) / 2
	offsetY := (imgH-graphH*scale)/2 + 20 // shift down for title

	tx := func(x float64) float64 { return (x-minX)*scale + offsetX }
	ty := func(y float64) float64 { return (y-minY)*scale + offsetY }

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`, int(imgW), int(imgH), int(imgW), int(imgH)))
	sb.WriteString(fmt.Sprintf(`<rect width="%d" height="%d" fill="#131214"/>`, int(imgW), int(imgH)))

	// Arrow marker
	sb.WriteString(`<defs><marker id="arrow" viewBox="0 0 10 10" refX="10" refY="5" markerWidth="6" markerHeight="6" orient="auto-start-reverse"><path d="M 0 0 L 10 5 L 0 10 z" fill="#555"/></marker></defs>`)

	// Edges
	for _, e := range edges {
		src, ok1 := posMap[e.SourceID]
		tgt, ok2 := posMap[e.TargetID]
		if !ok1 || !ok2 {
			continue
		}
		sb.WriteString(fmt.Sprintf(`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#555" stroke-width="1.5" marker-end="url(#arrow)"/>`,
			tx(src.PositionX), ty(src.PositionY), tx(tgt.PositionX), ty(tgt.PositionY)))
	}

	// GROUP nodes (background)
	for _, n := range nodes {
		if n.Type != model.NodeTypeGroup {
			continue
		}
		color := typeColors[n.Type]
		cx, cy := tx(n.PositionX), ty(n.PositionY)
		nw, nh := nodeSize(n)
		w, h := nw*scale, nh*scale
		sb.WriteString(fmt.Sprintf(`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="8" fill="none" stroke="%s" stroke-width="1.5" stroke-dasharray="6 3" opacity="0.6"/>`,
			cx-w/2, cy-h/2, w, h, color))
		title := truncate(n.Title, 20)
		sb.WriteString(fmt.Sprintf(`<text x="%.1f" y="%.1f" fill="%s" font-family="sans-serif" font-size="11" text-anchor="middle" opacity="0.7">%s</text>`,
			cx, cy-h/2+16, color, html.EscapeString(title)))
	}

	// Regular nodes
	for _, n := range nodes {
		if n.Type == model.NodeTypeGroup {
			continue
		}
		color := typeColors[n.Type]
		if color == "" {
			color = "#7ca3c4"
		}
		cx, cy := tx(n.PositionX), ty(n.PositionY)
		nw := 72.0 * scale
		nh := 36.0 * scale
		if nw < 40 {
			nw = 40
		}
		if nh < 20 {
			nh = 20
		}
		sb.WriteString(fmt.Sprintf(`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="6" fill="%s" opacity="0.85"/>`,
			cx-nw/2, cy-nh/2, nw, nh, color))
		title := truncate(n.Title, 12)
		fontSize := 11.0 * scale
		if fontSize < 8 {
			fontSize = 8
		}
		if fontSize > 13 {
			fontSize = 13
		}
		sb.WriteString(fmt.Sprintf(`<text x="%.1f" y="%.1f" fill="white" font-family="sans-serif" font-size="%.0f" text-anchor="middle" dominant-baseline="central">%s</text>`,
			cx, cy, fontSize, html.EscapeString(title)))
	}

	// Stats
	sb.WriteString(fmt.Sprintf(`<text x="20" y="28" fill="#c9a84c" font-family="sans-serif" font-size="14" font-weight="bold">%d nodes · %d edges</text>`,
		len(nodes), len(edges)))

	// Branding
	sb.WriteString(fmt.Sprintf(`<text x="%d" y="%d" fill="#555" font-family="sans-serif" font-size="10" text-anchor="end">Thask</text>`,
		int(imgW)-12, int(imgH)-12))

	sb.WriteString(`</svg>`)

	c.Response().Header().Set("Content-Type", "image/svg+xml")
	c.Response().Header().Set("Cache-Control", "public, max-age=300")
	return c.String(http.StatusOK, sb.String())
}

func nodeSize(n model.Node) (float64, float64) {
	if n.Type == model.NodeTypeGroup {
		w := 160.0
		h := 100.0
		if n.Width != nil {
			w = *n.Width
		}
		if n.Height != nil {
			h = *n.Height
		}
		return w, h
	}
	return 72, 36
}

func renderEmptySVG(name string, w, h float64) string {
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d">
		<rect width="%d" height="%d" fill="#131214"/>
		<text x="%.0f" y="%.0f" fill="#c9a84c" font-family="sans-serif" font-size="20" text-anchor="middle" font-weight="bold">%s</text>
		<text x="%.0f" y="%.0f" fill="#9e9c97" font-family="sans-serif" font-size="14" text-anchor="middle">No nodes yet</text>
	</svg>`, int(w), int(h), int(w), int(h), w/2, h/2-10, html.EscapeString(name), w/2, h/2+16)
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + "…"
}
