package main

import "testing"

func TestIntToBase58str(t *testing.T) {
	m := map[int]string{
		0: "1",
		1000: "if",
		1000000: "68go",
		1000000000: "2wngaj",
	}
	for i, str58 := range m {
		res := IntToBase58str(i)
		if res != str58 {
			t.Log("Input:", i,
				"Output:", res,
				"Expected output:", str58)
			t.Fail()
		}
	}
}
