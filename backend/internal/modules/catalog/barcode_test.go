package catalog

import "testing"

func TestValidateBarcodeAcceptsEAN13AndUPCA(t *testing.T) {
	for _, value := range []string{"4006381333931", "036000291452"} {
		t.Run(value, func(t *testing.T) {
			if !ValidateBarcode(value) {
				t.Fatalf("ValidateBarcode(%q) = false, want true", value)
			}
		})
	}
}

func TestValidateBarcodeRejectsInvalidValues(t *testing.T) {
	for _, value := range []string{"4006381333932", "036000291453", "40063813339A1", "", "03600029145", "12345678901234"} {
		t.Run(value, func(t *testing.T) {
			if ValidateBarcode(value) {
				t.Fatalf("ValidateBarcode(%q) = true, want false", value)
			}
		})
	}
}
