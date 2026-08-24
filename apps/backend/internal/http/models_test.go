package httpapi

import (
	"archive/zip"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestAnalyse3MFWithComponentsBuildTransformsAndUnits(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fixture.3mf")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	archive := zip.NewWriter(file)
	modelPart, err := archive.Create("3D/3dmodel.model")
	if err != nil {
		t.Fatal(err)
	}
	const model = `<?xml version="1.0" encoding="UTF-8"?>
<model xmlns="http://schemas.microsoft.com/3dmanufacturing/core/2015/02" unit="centimeter">
  <resources>
    <object id="1" type="model"><mesh>
      <vertices>
        <vertex x="0" y="0" z="0"/><vertex x="1" y="0" z="0"/>
        <vertex x="0" y="1" z="0"/><vertex x="0" y="0" z="1"/>
      </vertices>
      <triangles>
        <triangle v1="0" v2="2" v3="1"/><triangle v1="0" v2="1" v3="3"/>
        <triangle v1="1" v2="2" v3="3"/><triangle v1="2" v2="0" v3="3"/>
      </triangles>
    </mesh></object>
    <object id="2" type="model"><components>
      <component objectid="1" transform="1 0 0 0 1 0 0 0 1 0 0 0"/>
    </components></object>
  </resources>
  <build>
    <item objectid="2"/>
    <item objectid="2" transform="1 0 0 0 1 0 0 0 1 2 0 0"/>
  </build>
</model>`
	if _, err := modelPart.Write([]byte(model)); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	info, err := analyse3MF(path)
	if err != nil {
		t.Fatalf("analyse3MF() error = %v", err)
	}
	if info.X != 30 || info.Y != 10 || info.Z != 10 {
		t.Fatalf("dimensions = %.3f x %.3f x %.3f mm, want 30 x 10 x 10", info.X, info.Y, info.Z)
	}
	if info.Triangles != 8 {
		t.Fatalf("triangles = %d, want 8", info.Triangles)
	}
	if math.Abs(info.VolumeCM3-1.0/3.0) > 1e-9 {
		t.Fatalf("volume = %.12f cm3, want %.12f", info.VolumeCM3, 1.0/3.0)
	}
}

func TestParseThreeMFTransformRejectsMalformedValues(t *testing.T) {
	if _, err := parseThreeMFTransform("1 0 0"); err == nil {
		t.Fatal("parseThreeMFTransform() accepted a short transform")
	}
	if _, err := parseThreeMFTransform("1 0 0 0 1 0 0 0 1 0 NaN 0"); err == nil {
		t.Fatal("parseThreeMFTransform() accepted NaN")
	}
}
