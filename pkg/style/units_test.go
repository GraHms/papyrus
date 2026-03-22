package style

import (
	"math"
	"testing"
)

func TestParseLength(t *testing.T) {
	tests := []struct {
		input    string
		wantVal  float64
		wantUnit Unit
		wantErr  bool
	}{
		{"10pt", 10, UnitPt, false},
		{"12px", 12, UnitPx, false},
		{"20mm", 20, UnitMm, false},
		{"1.5em", 1.5, UnitEm, false},
		{"50%", 50, UnitPct, false},
		{"auto", 0, UnitAuto, false},
		{"", 0, 0, true},
		{"notanumber", 0, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			l, err := ParseLength(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseLength(%q): expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseLength(%q) error: %v", tt.input, err)
			}
			if l.Unit != tt.wantUnit {
				t.Errorf("ParseLength(%q).Unit = %v, want %v", tt.input, l.Unit, tt.wantUnit)
			}
			if math.Abs(l.Value-tt.wantVal) > 0.001 {
				t.Errorf("ParseLength(%q).Value = %v, want %v", tt.input, l.Value, tt.wantVal)
			}
		})
	}
}

func TestToPoints(t *testing.T) {
	const eps = 0.01
	tests := []struct {
		l       Length
		parent  float64
		font    float64
		root    float64
		dpi     float64
		wantPt  float64
	}{
		{Pt(10), 0, 10, 10, 96, 10},
		{Mm(25.4), 0, 10, 10, 96, 72},          // 1 inch = 72pt
		{Length{1, UnitIn}, 0, 10, 10, 96, 72},
		{Length{2, UnitEm}, 0, 12, 10, 96, 24},
		{Pct(50), 200, 10, 10, 96, 100},
		{Length{96, UnitPx}, 0, 10, 10, 96, 72}, // 96px @ 96dpi = 1in = 72pt
	}
	for _, tt := range tests {
		got := tt.l.ToPoints(tt.parent, tt.font, tt.root, tt.dpi)
		if math.Abs(got-tt.wantPt) > eps {
			t.Errorf("ToPoints(%+v) = %.4f, want %.4f", tt.l, got, tt.wantPt)
		}
	}
}

func TestParseColor(t *testing.T) {
	tests := []struct {
		input   string
		wantR   uint8
		wantG   uint8
		wantB   uint8
		wantA   uint8
		wantErr bool
	}{
		{"#000000", 0, 0, 0, 255, false},
		{"#ffffff", 255, 255, 255, 255, false},
		{"#fff", 255, 255, 255, 255, false},
		{"#e60000", 230, 0, 0, 255, false},
		{"rgb(255,0,0)", 255, 0, 0, 255, false},
		{"rgba(0,0,255,0.5)", 0, 0, 255, 127, false},
		{"black", 0, 0, 0, 255, false},
		{"white", 255, 255, 255, 255, false},
		{"transparent", 0, 0, 0, 0, false},
		{"notacolor", 0, 0, 0, 0, true},
		{"", 0, 0, 0, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			c, err := ParseColor(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseColor(%q): expected error", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseColor(%q) error: %v", tt.input, err)
			}
			if c.R != tt.wantR || c.G != tt.wantG || c.B != tt.wantB || c.A != tt.wantA {
				t.Errorf("ParseColor(%q) = {%d,%d,%d,%d}, want {%d,%d,%d,%d}",
					tt.input, c.R, c.G, c.B, c.A, tt.wantR, tt.wantG, tt.wantB, tt.wantA)
			}
		})
	}
}
