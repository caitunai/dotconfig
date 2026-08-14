package handler

import "testing"

const testConfigValuePath = "/service/host"

func TestConfigValueUpdatesAcceptsBatchAndLegacyForms(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		form configValuesPatchForm
		want int
	}{
		{
			name: "batch",
			form: configValuesPatchForm{Updates: []configValueUpdateForm{
				{Path: testConfigValuePath, Value: []byte(`"api"`)},
				{Path: "/service/port", Value: []byte(`443`)},
			}},
			want: 2,
		},
		{
			name: "legacy",
			form: configValuesPatchForm{Path: testConfigValuePath, Value: []byte(`"api"`)},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			updates, err := configValueUpdates(&tt.form)
			if err != nil || len(updates) != tt.want {
				t.Fatalf("configValueUpdates() count = %d, error = %v; want %d", len(updates), err, tt.want)
			}
		})
	}
}

func TestConfigValueUpdatesRejectsMixedForms(t *testing.T) {
	t.Parallel()

	form := &configValuesPatchForm{
		Path:    testConfigValuePath,
		Value:   []byte(`"api"`),
		Updates: []configValueUpdateForm{{Path: "/service/port", Value: []byte(`443`)}},
	}
	if _, err := configValueUpdates(form); err == nil {
		t.Fatal("configValueUpdates() error = nil, want mixed form rejection")
	}
}
