package main

import (
	"io/ioutil"
	"strings"
	"testing"
)

func TestFixtures(t *testing.T) {
	var err error

	var expectedSlice []string
	expectedMap := make(map[string]struct{})
	{
		bytes, err := ioutil.ReadFile("fixtures/errors.txt")
		if err != nil {
			t.Fatal(err)
		}

		expectedSlice = strings.Split(string(bytes), "\n")
		for _, expected := range expectedSlice {
			expectedMap[expected] = struct{}{}
		}
	}

	var actualSlice []string
	actualMap := make(map[string]struct{})
	{
		actualSlice, err = Lint("fixtures")
		if err != nil {
			t.Fatal(err)
		}

		for _, actual := range actualSlice {
			actualMap[actual] = struct{}{}
		}
	}

	for expected := range expectedMap {
		if _, ok := actualMap[expected]; !ok {
			t.Errorf("Expected error:\n%v", expected)
		}
	}

	for actual := range actualMap {
		if _, ok := expectedMap[actual]; !ok {
			t.Errorf("Did not expect error:\n%v", actual)
		}
	}

	for i := 0; i < len(actualSlice); i++ {
		if actualSlice[i] != expectedSlice[i] {
			t.Errorf("Errors are not ordered")
		}
	}
}
