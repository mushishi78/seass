package main

import (
	"io/ioutil"
	"strings"
	"testing"
)

func TestFixtures(t *testing.T) {
	expectedErrors := make(map[string]struct{})
	{
		bytes, err := ioutil.ReadFile("fixtures/errors.txt")
		if err != nil {
			t.Fatal(err)
		}

		for _, expected := range strings.Split(string(bytes), "\n") {
			expectedErrors[expected] = struct{}{}
		}
	}

	actualErrors, err := Lint("fixtures")

	if err != nil {
		t.Fatal(err)
	}

	for expected := range expectedErrors {
		if _, ok := actualErrors[expected]; !ok {
			t.Errorf("Expected error:\n%v", expected)
		}
	}

	for actual := range actualErrors {
		if _, ok := expectedErrors[actual]; !ok {
			t.Errorf("Did not expect error:\n%v", actual)
		}
	}

}
