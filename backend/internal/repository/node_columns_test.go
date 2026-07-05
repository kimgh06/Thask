package repository

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/thask/backend/internal/dbgen"
	"github.com/thask/backend/internal/model"
)

// TestNodeReadPaths_IncludeAllPersistedFields is the guard against
// `feedback_sql_scan_safety` regressions: any column persisted on the `nodes`
// table MUST appear in nodeColsWithCreator (the raw pgx SELECT list) so the
// dynamic filter path returns the same shape as the sqlc-generated queries.
//
// When you add a column to `nodes` and forget to append `n.<col>` here, this
// test fails and points at the exact missing column — no debugging required.
//
// The test only covers nodeColsWithCreator / scanNodesRaw. The six
// nodeFrom*Row converter funcs are covered by TestNodeConverters_PropagateAllFields
// below.
func TestNodeReadPaths_IncludeAllPersistedFields(t *testing.T) {
	// Reflect over model.Node and enumerate every db-tagged field that lives
	// on the nodes table. Skip `-` (unexported) and the JOIN-only column
	// creator_email — that one lives on users, not nodes.
	typ := reflect.TypeOf(model.Node{})
	for i := 0; i < typ.NumField(); i++ {
		tag := typ.Field(i).Tag.Get("db")
		if tag == "" || tag == "-" || tag == "creator_email" {
			continue
		}
		needle := "n." + tag
		if !strings.Contains(nodeColsWithCreator, needle) {
			t.Errorf(
				"nodeColsWithCreator is missing column %q (model.Node.%s). "+
					"Add it to backend/internal/repository/node.go AND to scanNodesRaw's Scan args.",
				needle, typ.Field(i).Name,
			)
		}
	}

	// Cross-check: scanNodesRaw's argument count must match the SELECT list
	// column count. If they drift, pgx scans the wrong field into the wrong
	// pointer and causes silent data corruption (or a runtime panic).
	//
	// Counting: SELECT items are separated by top-level commas, but
	// `COALESCE(u.email, '')` contains an internal comma we must not count.
	// The list has exactly one such expression today, and the invariant is
	// enforced separately here — if a future SELECT adds more paren-nested
	// commas, adjust nestedCommas below.
	const nestedCommas = 1
	selectCount := strings.Count(nodeColsWithCreator, ",") + 1 - nestedCommas
	if scanArgCount != selectCount {
		t.Errorf(
			"scanNodesRaw declares %d Scan args but nodeColsWithCreator SELECTs %d columns. "+
				"They MUST match. See backend/internal/repository/node.go scanNodesRaw.",
			scanArgCount, selectCount,
		)
	}
}

// TestNodeConverters_PropagateAllFields runs every nodeFrom*Row converter
// against a synthetic dbgen row whose fields are all populated with distinct
// sentinel values (via reflect). If a converter forgets to copy one of those
// fields into the resulting model.Node, that field stays at its Go zero value
// and this test fails naming the missing field.
//
// This is the safety-net for the converter-side half of
// feedback_sql_scan_safety — the SELECT-list guard above catches missed
// columns in the query, this guard catches missed mappings in the row-to-
// model bridge. v0.6.0 saw exactly this split-brain failure
// (`lifecycle_state` present in SELECT, absent from 6 converters).
//
// New converters MUST be added to the table below. Skipped fields are those
// that legitimately can't be populated by the given RETURNING clause
// (e.g. Create can't JOIN users to get creator_email).
func TestNodeConverters_PropagateAllFields(t *testing.T) {
	cases := []struct {
		name       string
		invoke     func() *model.Node
		skipFields []string // model.Node field names that CANNOT be populated by this path
	}{
		{
			name: "nodeFromCreateRow",
			invoke: func() *model.Node {
				var in dbgen.Node
				fillWithSentinels(reflect.ValueOf(&in).Elem())
				return nodeFromCreateRow(in)
			},
			// Create's RETURNING can't JOIN users; CreatorEmail is populated
			// on a subsequent FindByID.
			skipFields: []string{"CreatorEmail"},
		},
		{
			name: "nodeFromFindByIDRow",
			invoke: func() *model.Node {
				var in dbgen.NodeFindByIDRow
				fillWithSentinels(reflect.ValueOf(&in).Elem())
				return nodeFromFindByIDRow(in)
			},
		},
		{
			name: "nodeFromFindByIDsRow",
			invoke: func() *model.Node {
				var in dbgen.NodeFindByIDsRow
				fillWithSentinels(reflect.ValueOf(&in).Elem())
				return nodeFromFindByIDsRow(in)
			},
		},
		{
			name: "nodeFromFindByProjectIDSimpleRow",
			invoke: func() *model.Node {
				var in dbgen.NodeFindByProjectIDSimpleRow
				fillWithSentinels(reflect.ValueOf(&in).Elem())
				return nodeFromFindByProjectIDSimpleRow(in)
			},
		},
		{
			name: "nodeFromFindChangedSinceRow",
			invoke: func() *model.Node {
				var in dbgen.NodeFindChangedSinceRow
				fillWithSentinels(reflect.ValueOf(&in).Elem())
				return nodeFromFindChangedSinceRow(in)
			},
		},
		{
			name: "nodeFromFindFailOrBugRow",
			invoke: func() *model.Node {
				var in dbgen.NodeFindFailOrBugRow
				fillWithSentinels(reflect.ValueOf(&in).Elem())
				return nodeFromFindFailOrBugRow(in)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := tc.invoke()
			assertAllExportedFieldsNonZero(t, tc.name, reflect.ValueOf(out).Elem(), tc.skipFields...)
		})
	}
}

// fillWithSentinels sets every settable field of a struct value to a
// non-zero sentinel. Recurses into anonymous embedded structs. Pointer
// fields are allocated and their pointee is set. Slices are given a
// single non-zero element. time.Time is set to now. Skips fields we
// can't confidently populate (channels, funcs, maps of complex types).
func fillWithSentinels(v reflect.Value) {
	typ := v.Type()
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if !f.CanSet() {
			continue
		}
		setSentinel(f, typ.Field(i).Name)
	}
}

func setSentinel(f reflect.Value, name string) {
	t := f.Type()
	// time.Time — set to a fixed known instant.
	if t == reflect.TypeOf(time.Time{}) {
		f.Set(reflect.ValueOf(time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)))
		return
	}
	// json.RawMessage is []byte under the hood but we want valid JSON.
	if t == reflect.TypeOf(json.RawMessage(nil)) {
		f.SetBytes([]byte(`{"x":1}`))
		return
	}
	switch f.Kind() {
	case reflect.String:
		f.SetString("sentinel-" + name)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		f.SetInt(1)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		f.SetUint(1)
	case reflect.Float32, reflect.Float64:
		f.SetFloat(1.5)
	case reflect.Bool:
		f.SetBool(true)
	case reflect.Slice:
		// Special-case []byte (which json.RawMessage sits on top of).
		if t.Elem().Kind() == reflect.Uint8 {
			f.SetBytes([]byte("sentinel"))
			return
		}
		// Build a one-element slice with the element sentinel-populated.
		elem := reflect.New(t.Elem()).Elem()
		setSentinel(elem, name+"[0]")
		s := reflect.MakeSlice(t, 1, 1)
		s.Index(0).Set(elem)
		f.Set(s)
	case reflect.Ptr:
		p := reflect.New(t.Elem())
		setSentinel(p.Elem(), "*"+name)
		f.Set(p)
	case reflect.Interface:
		// model.Node.Metadata / FieldProvenance are `any` — a JSON blob
		// suffices as a non-zero payload.
		f.Set(reflect.ValueOf(json.RawMessage(`{"x":1}`)))
	case reflect.Struct:
		fillWithSentinels(f)
	default:
		// Skip channels, funcs, maps — none appear on dbgen row types.
	}
}

// assertAllExportedFieldsNonZero fails the test naming any exported field on
// the model.Node value that is still at its Go zero after conversion. Fields
// listed in `skip` are treated as intentionally unpopulated (documented per
// converter — e.g. Create's RETURNING can't JOIN users to fetch creator_email).
func assertAllExportedFieldsNonZero(t *testing.T, converter string, v reflect.Value, skip ...string) {
	skipSet := map[string]struct{}{}
	for _, s := range skip {
		skipSet[s] = struct{}{}
	}
	typ := v.Type()
	for i := 0; i < v.NumField(); i++ {
		name := typ.Field(i).Name
		if _, ok := skipSet[name]; ok {
			continue
		}
		if typ.Field(i).PkgPath != "" {
			continue // unexported
		}
		f := v.Field(i)
		if f.IsZero() {
			t.Errorf(
				"%s left model.Node.%s at zero value — converter is missing this field mapping. "+
					"See ~/.claude memory feedback_sql_scan_safety.",
				converter, name,
			)
		}
	}
}
