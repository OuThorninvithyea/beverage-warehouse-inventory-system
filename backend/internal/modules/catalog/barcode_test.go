package catalog

import "testing"

func TestValidBarcodeAcceptsEAN13AndUPCA(t *testing.T) {
	for _, value := range []string{"4006381333931", "036000291452"} {
		t.Run(value, func(t *testing.T) {
			if !ValidBarcode(value) {
				t.Fatalf("ValidBarcode(%q) = false, want true", value)
			}
		})
	}
}

func TestValidBarcodeRejectsInvalidValues(t *testing.T) {
	for _, value := range []string{"4006381333932", "036000291453", "40063813339A1", "", "03600029145", "12345678901234"} {
		t.Run(value, func(t *testing.T) {
			if ValidBarcode(value) {
				t.Fatalf("ValidBarcode(%q) = true, want false", value)
			}
		})
	}
}
