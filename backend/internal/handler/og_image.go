package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/thask/backend/internal/dto"
	mw "github.com/thask/backend/internal/middleware"
	"github.com/thask/backend/internal/model"
)

// ogPNGTimeout caps how long a social-media crawler will wait for og:image.
// Slack/Kakao/Twitter crawlers tend to give up around 5–10s, so we bound the
// PNG path well under the configured CaptureTimeout (default 30s) and fall
// back to inline SVG if the worker exceeds this.
const ogPNGTimeout = 8 * time.Second

var typeColors = map[model.NodeType]string{
	model.NodeTypeFlow:   "#e2b340",
	model.NodeTypeBranch: "#a78bfa",
	model.NodeTypeTask:   "#7ca3c4",
	model.NodeTypeBug:    "#e05252",
	model.NodeTypeAPI:    "#5ea87a",
	model.NodeTypeUI:     "#d4915a",
	model.NodeTypeGroup:  "#7c7570",
}

// OGImage renders a preview of the project graph for SNS share cards.
// Prefers PNG via the capture worker (better SNS compatibility); falls back
// to inline SVG if the worker is unconfigured, slow, or fails.
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

	if h.captureURL != "" {
		pngCtx, cancel := context.WithTimeout(ctx, ogPNGTimeout)
		defer cancel()
		if png, ct, _, err := h.capturePNG(pngCtx, nodes, edges, 1200, 630, 80, 2, true); err == nil {
			if ct == "" {
				ct = "image/png"
			}
			c.Response().Header().Set("Cache-Control", "public, max-age=300")
			return c.Blob(http.StatusOK, ct, png)
		}
		// fall through to inline SVG on worker error/timeout
	}

	svg := renderGraphSVG(nodes, edges, 1200, 630)
	c.Response().Header().Set("Content-Type", "image/svg+xml")
	c.Response().Header().Set("Cache-Control", "public, max-age=300")
	return c.String(http.StatusOK, svg)
}

// Capture renders a project graph image for external API and CLI use.
func (h *NodeHandler) Capture(c echo.Context) error {
	format := c.QueryParam("format")
	if format == "" {
		format = "png"
	}
	if format != "png" && format != "svg" {
		return c.JSON(http.StatusBadRequest, dto.Err("Only format=png or format=svg is supported"))
	}

	width := parseImageDimension(c.QueryParam("width"), 1600, 320, 4096)
	height := parseImageDimension(c.QueryParam("height"), 1000, 240, 4096)
	padding := parseImageDimension(c.QueryParam("padding"), 80, 0, 400)
	scale := parseImageDimension(c.QueryParam("scale"), 2, 1, 4)
	crop := parseBoolQuery(c.QueryParam("crop"), true)

	ctx := c.Request().Context()
	projectID := mw.ResolveProjectID(c)

	nodes, err := h.nodeRepo.FindByProjectID(ctx, projectID, nil, nil)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to fetch nodes"))
	}

	edges, err := h.edgeRepo.FindByProjectID(ctx, projectID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, dto.Err("Failed to fetch edges"))
	}

	if format == "svg" {
		svg := renderGraphSVG(nodes, edges, float64(width), float64(height))
		c.Response().Header().Set("Content-Type", "image/svg+xml")
		c.Response().Header().Set("Cache-Control", "no-store")
		return c.String(http.StatusOK, svg)
	}

	if h.captureURL == "" {
		return c.JSON(http.StatusServiceUnavailable, dto.Err("Capture worker is not configured"))
	}
	image, contentType, metrics, err := h.capturePNG(ctx, nodes, edges, width, height, padding, scale, crop)
	if err != nil {
		return c.JSON(http.StatusBadGateway, dto.Err(err.Error()))
	}
	if contentType == "" {
		contentType = "image/png"
	}
	c.Response().Header().Set("Cache-Control", "no-store")
	if metrics != "" {
		c.Response().Header().Set("X-Thask-Capture-Metrics", metrics)
	}
	return c.Blob(http.StatusOK, contentType, image)
}

func (h *NodeHandler) capturePNG(ctx context.Context, nodes []model.Node, edges []model.Edge, width, height, padding, scale int, crop bool) ([]byte, string, string, error) {
	endpoint, err := url.JoinPath(strings.TrimRight(h.captureURL, "/"), "/capture/graph")
	if err != nil {
		return nil, "", "", fmt.Errorf("invalid capture worker URL")
	}
	payload := map[string]any{
		"nodes":   nodes,
		"edges":   edges,
		"width":   width,
		"height":  height,
		"padding": padding,
		"scale":   scale,
		"crop":    crop,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to build capture request: %w", err)
	}
	reqCtx, cancel := context.WithTimeout(ctx, h.captureTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to create capture request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "image/png")
	if h.captureSecret != "" {
		req.Header.Set("X-Thask-Capture-Secret", h.captureSecret)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", "", fmt.Errorf("capture worker unavailable: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to read capture response: %w", err)
	}
	if resp.StatusCode >= 400 {
		var errResp struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(respBody, &errResp) == nil && errResp.Error != "" {
			return nil, "", "", fmt.Errorf("capture failed: %s", errResp.Error)
		}
		return nil, "", "", fmt.Errorf("capture failed: HTTP %d", resp.StatusCode)
	}
	return respBody, resp.Header.Get("Content-Type"), resp.Header.Get("X-Thask-Capture-Metrics"), nil
}

func renderGraphSVG(nodes []model.Node, edges []model.Edge, imgW, imgH float64) string {
	if len(nodes) == 0 {
		return renderEmptySVG("Empty Graph", imgW, imgH)
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

	return sb.String()
}

func parseImageDimension(raw string, fallback, minValue, maxValue int) int {
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	if v < minValue {
		return minValue
	}
	if v > maxValue {
		return maxValue
	}
	return v
}

func parseBoolQuery(raw string, fallback bool) bool {
	if raw == "" {
		return fallback
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
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
