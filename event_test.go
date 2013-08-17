package godj

import (
	"bytes"
	"testing"
)

func TestSerializeEvent(t *testing.T) {
	expected := "-1\x00mkdir\x00/var/lib/docker\x001"
	e := &Event{
		Id:          1,
		Description: []string{"mkdir", "/var/lib/docker"},
		Overlap:     -1,
		isComplete:  true,
	}

	actual := e.Serialize()
	if actual != expected {
		t.Logf("Expected: %s, Actual: %s", expected, actual)
		t.Fail()
	}
}

func TestDeserializeEvent(t *testing.T) {
	expected := "-1\x00mkdir\x00/var/lib/docker\x001"

	r := bytes.NewReader([]byte(expected))

	e, err := DeserializeEvent(1, r)
	if err != nil {
		t.Fatal(err)
	}

	if e == nil {
		t.Log("Event should not be nil")
		t.FailNow()
	}

	if e.Id != 1 {
		t.Fail()
	}
	if e.Overlap != -1 {
		t.Fail()
	}
	if len(e.Description) != 2 {
		t.Fail()
	}
	if e.Description[0] != "mkdir" {
		t.Fail()
	}
	if e.Description[1] != "/var/lib/docker" {
		t.Fail()
	}
}
