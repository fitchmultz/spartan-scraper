package mcp

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/scheduler"
)

func TestExportScheduleToolsInToolsList(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	toolMap := make(map[string]tool)
	for _, tool := range srv.toolsList() {
		toolMap[tool.Name] = tool
	}
	for _, name := range []string{
		"export_schedule_list",
		"export_schedule_get",
		"export_schedule_create",
		"export_schedule_update",
		"export_schedule_delete",
		"export_schedule_history",
	} {
		if _, ok := toolMap[name]; !ok {
			t.Fatalf("expected tool %s in toolsList", name)
		}
	}
}

func TestHandleExportScheduleLifecycle(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	ctx := context.Background()

	createBase := map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name": "export_schedule_create",
			"arguments": map[string]interface{}{
				"name": "Projected Export",
				"filters": map[string]interface{}{
					"job_kinds": []string{"scrape"},
				},
				"export": map[string]interface{}{
					"format":           "csv",
					"destination_type": "local",
					"transform": map[string]interface{}{
						"expression": "{title: title, url: url}",
						"language":   "jmespath",
					},
				},
			},
		}),
	}

	createdResult, err := srv.handleToolCall(ctx, createBase)
	if err != nil {
		t.Fatalf("export_schedule_create failed: %v", err)
	}
	createdSchedule, ok := createdResult.(*scheduler.ExportSchedule)
	if !ok {
		t.Fatalf("expected created schedule, got %#v", createdResult)
	}
	if createdSchedule.Export.Transform.Expression != "{title: title, url: url}" {
		t.Fatalf("unexpected transform: %#v", createdSchedule.Export.Transform)
	}
	if createdSchedule.Export.LocalPath != "exports/{kind}/{job_id}.{format}" {
		t.Fatalf("unexpected normalized local path: %q", createdSchedule.Export.LocalPath)
	}

	listBase := map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name":      "export_schedule_list",
			"arguments": map[string]interface{}{},
		}),
	}
	listedResult, err := srv.handleToolCall(ctx, listBase)
	if err != nil {
		t.Fatalf("export_schedule_list failed: %v", err)
	}
	listedMap, ok := listedResult.(map[string]interface{})
	if !ok {
		t.Fatalf("expected list map, got %#v", listedResult)
	}
	schedules, ok := listedMap["schedules"].([]scheduler.ExportSchedule)
	if !ok || len(schedules) != 1 {
		t.Fatalf("unexpected schedules payload: %#v", listedMap["schedules"])
	}

	updateBase := map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name": "export_schedule_update",
			"arguments": map[string]interface{}{
				"id":      createdSchedule.ID,
				"name":    "Projected Export Updated",
				"enabled": false,
				"filters": map[string]interface{}{
					"job_kinds": []string{"scrape"},
				},
				"export": map[string]interface{}{
					"format":           "json",
					"destination_type": "local",
					"transform": map[string]interface{}{
						"expression": "{title: title}",
						"language":   "jmespath",
					},
				},
			},
		}),
	}
	updatedResult, err := srv.handleToolCall(ctx, updateBase)
	if err != nil {
		t.Fatalf("export_schedule_update failed: %v", err)
	}
	updatedSchedule, ok := updatedResult.(*scheduler.ExportSchedule)
	if !ok {
		t.Fatalf("expected updated schedule, got %#v", updatedResult)
	}
	if updatedSchedule.Enabled {
		t.Fatalf("expected updated schedule to be disabled")
	}
	if updatedSchedule.Export.Transform.Expression != "{title: title}" {
		t.Fatalf("unexpected updated transform: %#v", updatedSchedule.Export.Transform)
	}

	getBase := map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name": "export_schedule_get",
			"arguments": map[string]interface{}{
				"id": createdSchedule.ID,
			},
		}),
	}
	gotResult, err := srv.handleToolCall(ctx, getBase)
	if err != nil {
		t.Fatalf("export_schedule_get failed: %v", err)
	}
	gotSchedule, ok := gotResult.(*scheduler.ExportSchedule)
	if !ok {
		t.Fatalf("expected get schedule, got %#v", gotResult)
	}
	if gotSchedule.Name != "Projected Export Updated" {
		t.Fatalf("unexpected schedule name: %q", gotSchedule.Name)
	}

	historyBase := map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name": "export_schedule_history",
			"arguments": map[string]interface{}{
				"id": createdSchedule.ID,
			},
		}),
	}
	historyResult, err := srv.handleToolCall(ctx, historyBase)
	if err != nil {
		t.Fatalf("export_schedule_history failed: %v", err)
	}
	payload, err := json.Marshal(historyResult)
	if err != nil {
		t.Fatalf("marshal history result: %v", err)
	}
	var historyMap map[string]interface{}
	if err := json.Unmarshal(payload, &historyMap); err != nil {
		t.Fatalf("decode history result: %v", err)
	}
	if total, ok := historyMap["total"].(float64); !ok || total != 0 {
		t.Fatalf("unexpected history total: %#v", historyMap["total"])
	}

	deleteBase := map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name": "export_schedule_delete",
			"arguments": map[string]interface{}{
				"id": createdSchedule.ID,
			},
		}),
	}
	if _, err := srv.handleToolCall(ctx, deleteBase); err != nil {
		t.Fatalf("export_schedule_delete failed: %v", err)
	}
}
