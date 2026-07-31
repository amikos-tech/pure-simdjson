package purejson

import "testing"

func TestValidateUTF8(t *testing.T) {
	testCases := []struct {
		name string
		data []byte
		want bool
	}{
		{name: "ascii", data: []byte("plain text"), want: true},
		{name: "multibyte", data: []byte("Здравей"), want: true},
		{name: "invalid continuation", data: []byte{0x80}, want: false},
		{name: "empty", data: nil, want: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ValidateUTF8(tc.data)
			if err != nil {
				t.Fatalf("ValidateUTF8() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("ValidateUTF8() = %t, want %t", got, tc.want)
			}
		})
	}
}
