package main

// Demo fixtures for local development and manual QA. Everything here is
// fictional: the warehouses, the beverages and the GS1 prefixes are chosen so
// the dataset can never collide with a real trade item.
//
// Product barcodes use the 884 (Cambodia) EAN-13 prefix plus a few UPC-A
// values for imported goods, so both supported formats are exercised.
// Location barcodes use the 20x restricted-circulation prefix, which is
// reserved for internal use. Check digits are computed by ean13 and upcA
// rather than typed, so they cannot drift; demo_test.go verifies the result
// against the same validator the API uses.
//
// The dataset is deliberately larger than the 20-row default page size so
// pagination, filtering and empty states are all reachable, and it includes
// the awkward cases on purpose: inactive rows, products without a barcode or
// category, products that are not lot tracked, stock in several locations,
// reserved quantities, and lots that are already expired.
//
// Flags are negative (inactive, untracked) so the common case is the zero
// value and a forgotten field cannot silently deactivate a row.

type demoWarehouse struct {
	code     string
	name     string
	address  string
	inactive bool
}

type demoLocation struct {
	warehouse   string
	code        string
	zone        string
	aisle       string
	rack        string
	shelf       string
	barcode     string
	notPickable bool
	inactive    bool
}

type demoUser struct {
	email     string
	fullName  string
	role      string
	warehouse string // empty means the user is not scoped to one warehouse
	inactive  bool
}

type demoCategory struct {
	name     string
	parent   string // empty for a root category
	inactive bool
}

type demoProduct struct {
	sku       string
	name      string
	category  string // empty means uncategorised
	unit      string
	barcode   string // empty means the product has no barcode
	untracked bool   // not lot tracked
	inactive  bool
}

type demoLot struct {
	sku             string
	number          string
	expiresInDays   int // negative is already expired
	noExpiry        bool
	receivedDaysAgo int
}

// demoMovement is one stock event. Balances and FIFO cost layers are derived
// from these events instead of being written by hand, so the seeded ledger
// stays internally consistent.
type demoMovement struct {
	kind      string // receive, pick, transfer or adjust
	sku       string
	lot       string // empty for products that are not lot tracked
	from      string // "warehouse/location", empty for receive and positive adjust
	to        string // "warehouse/location", empty for pick and negative adjust
	quantity  string
	unitCost  string // receive and positive adjust only
	daysAgo   int
	reference string
	notes     string
	actor     string // seeded user email
}

const (
	warehousePhnomPenh  = "PP-CENTRAL"
	warehouseSiemReap   = "SR-DEPOT"
	warehouseBattambang = "BB-HUB"
	warehouseRetired    = "KP-CLOSED"

	userManagerPP = "manager@bwims.local"
	userManagerSR = "manager.sr@bwims.local"
	userManagerBB = "manager.bb@bwims.local"
	userPickerPP  = "picker@bwims.local"
	userPickerPP2 = "picker2@bwims.local"
	userPickerSR  = "picker.sr@bwims.local"
	userPickerBB  = "picker.bb@bwims.local"
	userViewer    = "viewer@bwims.local"

	demoReference = "SEED-"
)

// checkDigit implements the shared EAN-13/UPC-A modulo-10 rule: weight the
// body from the right, alternating 3 and 1.
func checkDigit(body string) string {
	total := 0
	for index, char := range body {
		digit := int(char - '0')
		if (len(body)-1-index)%2 == 0 {
			total += digit * 3
			continue
		}
		total += digit
	}
	return string(rune('0' + (10-(total%10))%10))
}

func ean13(body12 string) string { return body12 + checkDigit(body12) }
func upcA(body11 string) string  { return body11 + checkDigit(body11) }

var demoWarehouses = []demoWarehouse{
	{code: warehousePhnomPenh, name: "Phnom Penh Central DC", address: "Street 271, Sen Sok, Phnom Penh"},
	{code: warehouseSiemReap, name: "Siem Reap Depot", address: "National Road 6, Siem Reap"},
	{code: warehouseBattambang, name: "Battambang Hub", address: "National Road 5, Battambang"},
	{code: warehouseRetired, name: "Kampot Depot (closed)", address: "Street 724, Kampot", inactive: true},
}

var demoLocations = []demoLocation{
	// Phnom Penh: the deep warehouse, with every location type.
	{warehouse: warehousePhnomPenh, code: "RECV-DOCK", zone: "RECEIVING", barcode: ean13("200100019001"), notPickable: true},
	{warehouse: warehousePhnomPenh, code: "SHIP-DOCK", zone: "SHIPPING", barcode: ean13("200100019003"), notPickable: true},
	{warehouse: warehousePhnomPenh, code: "QUAR-01", zone: "QUARANTINE", barcode: ean13("200100019002"), notPickable: true},
	{warehouse: warehousePhnomPenh, code: "A-01-01", zone: "AMBIENT", aisle: "A", rack: "01", shelf: "01", barcode: ean13("200100010101")},
	{warehouse: warehousePhnomPenh, code: "A-01-02", zone: "AMBIENT", aisle: "A", rack: "01", shelf: "02", barcode: ean13("200100010102")},
	{warehouse: warehousePhnomPenh, code: "A-02-01", zone: "AMBIENT", aisle: "A", rack: "02", shelf: "01", barcode: ean13("200100010201")},
	{warehouse: warehousePhnomPenh, code: "B-01-01", zone: "AMBIENT", aisle: "B", rack: "01", shelf: "01", barcode: ean13("200100010301")},
	{warehouse: warehousePhnomPenh, code: "COLD-01", zone: "CHILLED", aisle: "C", rack: "01", shelf: "01", barcode: ean13("200100010401")},
	{warehouse: warehousePhnomPenh, code: "COLD-02", zone: "CHILLED", aisle: "C", rack: "01", shelf: "02", barcode: ean13("200100010402")},
	{warehouse: warehousePhnomPenh, code: "OLD-01", zone: "AMBIENT", aisle: "Z", rack: "01", shelf: "01", barcode: ean13("200100019900"), inactive: true},

	// Siem Reap: a mid-size depot.
	{warehouse: warehouseSiemReap, code: "RECV-DOCK", zone: "RECEIVING", barcode: ean13("200200019001"), notPickable: true},
	{warehouse: warehouseSiemReap, code: "A-01-01", zone: "AMBIENT", aisle: "A", rack: "01", shelf: "01", barcode: ean13("200200010101")},
	{warehouse: warehouseSiemReap, code: "A-01-02", zone: "AMBIENT", aisle: "A", rack: "01", shelf: "02", barcode: ean13("200200010102")},
	{warehouse: warehouseSiemReap, code: "COLD-01", zone: "CHILLED", aisle: "C", rack: "01", shelf: "01", barcode: ean13("200200010301")},

	// Battambang: a small hub, useful for an almost-empty warehouse.
	{warehouse: warehouseBattambang, code: "RECV-DOCK", zone: "RECEIVING", barcode: ean13("200300019001"), notPickable: true},
	{warehouse: warehouseBattambang, code: "A-01-01", zone: "AMBIENT", aisle: "A", rack: "01", shelf: "01", barcode: ean13("200300010101")},

	// A location in the closed warehouse, so deactivation is visible.
	{warehouse: warehouseRetired, code: "A-01-01", zone: "AMBIENT", aisle: "A", rack: "01", shelf: "01", barcode: ean13("200400010101"), inactive: true},
}

var demoUsers = []demoUser{
	{email: userManagerPP, fullName: "Demo Manager Phnom Penh", role: "warehouse_manager", warehouse: warehousePhnomPenh},
	{email: userManagerSR, fullName: "Demo Manager Siem Reap", role: "warehouse_manager", warehouse: warehouseSiemReap},
	{email: userManagerBB, fullName: "Demo Manager Battambang", role: "warehouse_manager", warehouse: warehouseBattambang},
	{email: userPickerPP, fullName: "Demo Picker Phnom Penh", role: "picker", warehouse: warehousePhnomPenh},
	{email: userPickerPP2, fullName: "Demo Picker Phnom Penh Night Shift", role: "picker", warehouse: warehousePhnomPenh},
	{email: userPickerSR, fullName: "Demo Picker Siem Reap", role: "picker", warehouse: warehouseSiemReap},
	{email: userPickerBB, fullName: "Demo Picker Battambang", role: "picker", warehouse: warehouseBattambang},
	{email: userViewer, fullName: "Demo Viewer", role: "viewer"},
	{email: "auditor@bwims.local", fullName: "Demo Auditor", role: "viewer", warehouse: warehousePhnomPenh},
	{email: "former.picker@bwims.local", fullName: "Former Picker (deactivated)", role: "picker", warehouse: warehousePhnomPenh, inactive: true},
	{email: "former.manager@bwims.local", fullName: "Former Manager (deactivated)", role: "warehouse_manager", warehouse: warehouseSiemReap, inactive: true},
}

var demoCategories = []demoCategory{
	{name: "Beverages"},
	{name: "Carbonated Soft Drinks", parent: "Beverages"},
	{name: "Water", parent: "Beverages"},
	{name: "Juice", parent: "Beverages"},
	{name: "Energy Drinks", parent: "Beverages"},
	{name: "Beer", parent: "Beverages"},
	{name: "Wine and Spirits", parent: "Beverages"},
	{name: "Tea and Coffee", parent: "Beverages"},
	{name: "Dairy", parent: "Beverages"},
	{name: "Packaging"},
	{name: "Crates and Pallets", parent: "Packaging"},
	{name: "Cleaning Supplies"},
	{name: "Seasonal Promotions", inactive: true},
}

// 42 products: more than two default pages, covering every catalog edge case.
var demoProducts = []demoProduct{
	// Carbonated soft drinks
	{sku: "BEV-COLA-330", name: "Angkor Cola Can 330 ml", category: "Carbonated Soft Drinks", unit: "case", barcode: ean13("884100000101")},
	{sku: "BEV-COLA-1500", name: "Angkor Cola PET 1.5 L", category: "Carbonated Soft Drinks", unit: "case", barcode: ean13("884100000102")},
	{sku: "BEV-COLA-DIET-330", name: "Angkor Cola Zero Can 330 ml", category: "Carbonated Soft Drinks", unit: "case", barcode: ean13("884100000103")},
	{sku: "BEV-LEMON-330", name: "Mekong Lemon Soda Can 330 ml", category: "Carbonated Soft Drinks", unit: "case", barcode: ean13("884100000104")},
	{sku: "BEV-ORANGE-330", name: "Mekong Orange Soda Can 330 ml", category: "Carbonated Soft Drinks", unit: "case", barcode: ean13("884100000105")},
	{sku: "BEV-GINGER-330", name: "Bassac Ginger Ale Can 330 ml", category: "Carbonated Soft Drinks", unit: "case", barcode: ean13("884100000106")},
	{sku: "BEV-TONIC-200", name: "Bassac Tonic Water 200 ml", category: "Carbonated Soft Drinks", unit: "case", barcode: ean13("884100000107")},
	{sku: "BEV-SODA-1500", name: "Bassac Soda Water PET 1.5 L", category: "Carbonated Soft Drinks", unit: "case", barcode: ean13("884100000108")},

	// Water
	{sku: "WTR-STILL-500", name: "Kulen Still Water 500 ml", category: "Water", unit: "case", barcode: ean13("884100000201")},
	{sku: "WTR-STILL-1500", name: "Kulen Still Water 1.5 L", category: "Water", unit: "case", barcode: ean13("884100000202")},
	{sku: "WTR-SPARK-750", name: "Kulen Sparkling Water 750 ml", category: "Water", unit: "case", barcode: ean13("884100000203")},
	{sku: "WTR-MINERAL-330", name: "Bokor Mineral Water 330 ml", category: "Water", unit: "case", barcode: ean13("884100000204")},
	{sku: "WTR-GALLON-19", name: "Kulen Water Gallon 19 L", category: "Water", unit: "bottle", barcode: ean13("884100000205")},

	// Juice
	{sku: "JUI-ORNG-1000", name: "Tonle Orange Juice 1 L", category: "Juice", unit: "case", barcode: ean13("884100000301")},
	{sku: "JUI-MANGO-250", name: "Tonle Mango Nectar 250 ml", category: "Juice", unit: "case", barcode: ean13("884100000302")},
	{sku: "JUI-APPLE-1000", name: "Tonle Apple Juice 1 L", category: "Juice", unit: "case", barcode: ean13("884100000303")},
	{sku: "JUI-COCO-330", name: "Kampot Coconut Water 330 ml", category: "Juice", unit: "case", barcode: ean13("884100000304")},
	{sku: "JUI-LYCHEE-250", name: "Tonle Lychee Nectar 250 ml", category: "Juice", unit: "case", barcode: ean13("884100000305")},

	// Energy drinks
	{sku: "ENG-BOOST-250", name: "Bayon Boost Energy 250 ml", category: "Energy Drinks", unit: "case", barcode: ean13("884100000401")},
	{sku: "ENG-BOOST-SF-250", name: "Bayon Boost Sugar Free 250 ml", category: "Energy Drinks", unit: "case", barcode: ean13("884100000402")},
	{sku: "ENG-SPORT-500", name: "Bayon Sport Isotonic 500 ml", category: "Energy Drinks", unit: "case", barcode: ean13("884100000403")},

	// Beer
	{sku: "BEE-LAGER-330", name: "Riverside Lager Can 330 ml", category: "Beer", unit: "case", barcode: ean13("884100000501")},
	{sku: "BEE-LAGER-640", name: "Riverside Lager Bottle 640 ml", category: "Beer", unit: "crate", barcode: ean13("884100000502")},
	{sku: "BEE-STOUT-330", name: "Riverside Stout Can 330 ml", category: "Beer", unit: "case", barcode: ean13("884100000503")},
	{sku: "BEE-IPA-330", name: "Riverside IPA Can 330 ml", category: "Beer", unit: "case", barcode: ean13("884100000504")},
	{sku: "BEE-LIGHT-330", name: "Riverside Light Can 330 ml", category: "Beer", unit: "case", barcode: ean13("884100000505")},

	// Wine and spirits
	{sku: "WIN-RED-750", name: "Imported Red Wine 750 ml", category: "Wine and Spirits", unit: "case", barcode: ean13("884100000601")},
	{sku: "WIN-WHITE-750", name: "Imported White Wine 750 ml", category: "Wine and Spirits", unit: "case", barcode: ean13("884100000602")},
	{sku: "SPI-RUM-700", name: "Kampong Rum 700 ml", category: "Wine and Spirits", unit: "case", barcode: ean13("884100000603")},

	// Tea and coffee
	{sku: "COF-LATTE-240", name: "Phnom Coffee Latte Can 240 ml", category: "Tea and Coffee", unit: "case", barcode: ean13("884100000701")},
	{sku: "COF-BLACK-240", name: "Phnom Coffee Black Can 240 ml", category: "Tea and Coffee", unit: "case", barcode: ean13("884100000702")},
	{sku: "TEA-GREEN-500", name: "Angkor Green Tea 500 ml", category: "Tea and Coffee", unit: "case", barcode: ean13("884100000703")},
	{sku: "TEA-LEMON-500", name: "Angkor Lemon Tea 500 ml", category: "Tea and Coffee", unit: "case", barcode: ean13("884100000704")},

	// Dairy
	{sku: "DRY-MILK-200", name: "Siem Dairy Milk 200 ml", category: "Dairy", unit: "case", barcode: ean13("884100000801")},
	{sku: "DRY-YOG-180", name: "Siem Dairy Drinking Yoghurt 180 ml", category: "Dairy", unit: "case", barcode: ean13("884100000802")},

	// Imported goods carry UPC-A instead of EAN-13.
	{sku: "IMP-COLA-355", name: "Imported Cola Can 355 ml", category: "Carbonated Soft Drinks", unit: "case", barcode: upcA("04900001234")},
	{sku: "IMP-ROOT-355", name: "Imported Root Beer Can 355 ml", category: "Carbonated Soft Drinks", unit: "case", barcode: upcA("01200001234")},

	// Deliberate edge cases.
	{sku: "PKG-CRATE-24", name: "Returnable Crate 24 Slot", category: "Crates and Pallets", unit: "crate", barcode: ean13("884100000901"), untracked: true},
	{sku: "PKG-PALLET-EU", name: "Euro Pallet", category: "Crates and Pallets", unit: "pallet", untracked: true},
	{sku: "CLN-SANITIZER-5L", name: "Line Sanitizer 5 L", category: "Cleaning Supplies", unit: "bottle", barcode: ean13("884100000902")},
	{sku: "MSC-UNSORTED-001", name: "Unsorted Returns Pallet", unit: "pallet", untracked: true},
	{sku: "BEV-RETIRED-330", name: "Discontinued Cola Can 330 ml", category: "Carbonated Soft Drinks", unit: "case", barcode: ean13("884100000903"), inactive: true},
	{sku: "BEE-SEASONAL-330", name: "Seasonal Festival Beer 330 ml", category: "Seasonal Promotions", unit: "case", inactive: true},
}

// Curated lots drive the expiry scenarios. Everything else gets a lot from
// generateLots so each tracked product has stock with a sensible shelf life.
var curatedLots = []demoLot{
	// Two lots of one product in one location: the FEFO demonstration.
	{sku: "BEV-COLA-330", number: "L-COLA-2512", expiresInDays: 60, receivedDaysAgo: 120},
	{sku: "BEV-COLA-330", number: "L-COLA-2601", expiresInDays: 180, receivedDaysAgo: 40},

	// Already expired, in three different warehouses.
	{sku: "JUI-ORNG-1000", number: "L-ORNG-EXPIRED", expiresInDays: -5, receivedDaysAgo: 95},
	{sku: "DRY-YOG-180", number: "L-YOG-EXPIRED", expiresInDays: -12, receivedDaysAgo: 60},
	{sku: "JUI-COCO-330", number: "L-COCO-EXPIRED", expiresInDays: -2, receivedDaysAgo: 70},

	// Expiring inside a week, then a month.
	{sku: "DRY-MILK-200", number: "L-MILK-2607", expiresInDays: 7, receivedDaysAgo: 14},
	{sku: "COF-LATTE-240", number: "L-LATTE-2607", expiresInDays: 12, receivedDaysAgo: 20},
	{sku: "JUI-ORNG-1000", number: "L-ORNG-2606", expiresInDays: 21, receivedDaysAgo: 10},
	{sku: "JUI-LYCHEE-250", number: "L-LYCHEE-2608", expiresInDays: 28, receivedDaysAgo: 18},
	{sku: "TEA-GREEN-500", number: "L-GREEN-2609", expiresInDays: 45, receivedDaysAgo: 30},

	// A lot with no expiration date at all.
	{sku: "SPI-RUM-700", number: "L-RUM-NOEXP", noExpiry: true, receivedDaysAgo: 150},
}

// Curated movements carry the scenarios worth reading in the movement
// history. generateMovements adds the bulk receives and picks around them.
var curatedMovements = []demoMovement{
	{kind: "receive", sku: "BEV-COLA-330", lot: "L-COLA-2512", to: "PP-CENTRAL/A-01-01", quantity: "240", unitCost: "9.20", daysAgo: 120, reference: "SEED-RCV-1001", actor: userManagerPP},
	{kind: "receive", sku: "JUI-ORNG-1000", lot: "L-ORNG-EXPIRED", to: "PP-CENTRAL/COLD-01", quantity: "120", unitCost: "14.50", daysAgo: 95, reference: "SEED-RCV-1002", actor: userManagerPP},
	{kind: "receive", sku: "JUI-COCO-330", lot: "L-COCO-EXPIRED", to: "SR-DEPOT/COLD-01", quantity: "90", unitCost: "11.80", daysAgo: 70, reference: "SEED-RCV-1003", actor: userManagerSR},
	{kind: "receive", sku: "DRY-YOG-180", lot: "L-YOG-EXPIRED", to: "PP-CENTRAL/COLD-02", quantity: "60", unitCost: "13.20", daysAgo: 60, reference: "SEED-RCV-1004", actor: userManagerPP},
	{kind: "receive", sku: "SPI-RUM-700", lot: "L-RUM-NOEXP", to: "PP-CENTRAL/B-01-01", quantity: "48", unitCost: "42.00", daysAgo: 150, reference: "SEED-RCV-1005", actor: userManagerPP},
	{kind: "receive", sku: "BEV-COLA-330", lot: "L-COLA-2601", to: "PP-CENTRAL/A-01-01", quantity: "300", unitCost: "9.65", daysAgo: 40, reference: "SEED-RCV-1006", actor: userManagerPP},
	{kind: "receive", sku: "TEA-GREEN-500", lot: "L-GREEN-2609", to: "PP-CENTRAL/A-02-01", quantity: "180", unitCost: "8.40", daysAgo: 30, reference: "SEED-RCV-1007", actor: userManagerPP},
	{kind: "receive", sku: "COF-LATTE-240", lot: "L-LATTE-2607", to: "PP-CENTRAL/COLD-01", quantity: "96", unitCost: "15.20", daysAgo: 20, reference: "SEED-RCV-1008", actor: userManagerPP},
	{kind: "receive", sku: "JUI-LYCHEE-250", lot: "L-LYCHEE-2608", to: "PP-CENTRAL/A-01-02", quantity: "144", unitCost: "10.90", daysAgo: 18, reference: "SEED-RCV-1009", actor: userManagerPP},
	{kind: "receive", sku: "DRY-MILK-200", lot: "L-MILK-2607", to: "PP-CENTRAL/COLD-01", quantity: "240", unitCost: "11.40", daysAgo: 14, reference: "SEED-RCV-1010", actor: userManagerPP},
	{kind: "receive", sku: "JUI-ORNG-1000", lot: "L-ORNG-2606", to: "PP-CENTRAL/COLD-01", quantity: "180", unitCost: "14.95", daysAgo: 10, reference: "SEED-RCV-1011", actor: userManagerPP},

	// FEFO: the earlier-expiring cola lot is drawn first.
	{kind: "pick", sku: "BEV-COLA-330", lot: "L-COLA-2512", from: "PP-CENTRAL/A-01-01", quantity: "60", daysAgo: 12, reference: "SEED-PCK-2001", notes: "FEFO drew the earliest expiring cola lot", actor: userPickerPP},

	// Cross-warehouse transfer: moves the cost layer as well as the stock.
	{kind: "transfer", sku: "BEV-COLA-330", lot: "L-COLA-2601", from: "PP-CENTRAL/A-01-01", to: "SR-DEPOT/A-01-01", quantity: "120", daysAgo: 7, reference: "SEED-TRF-3001", notes: "Interwarehouse replenishment", actor: userManagerPP},

	// Adjustments, both directions.
	{kind: "adjust", sku: "TEA-GREEN-500", lot: "L-GREEN-2609", from: "PP-CENTRAL/A-02-01", quantity: "6", daysAgo: 4, reference: "SEED-ADJ-4001", notes: "Cycle count: damaged in handling", actor: userManagerPP},
	{kind: "adjust", sku: "DRY-MILK-200", lot: "L-MILK-2607", to: "PP-CENTRAL/COLD-01", quantity: "12", unitCost: "11.40", daysAgo: 2, reference: "SEED-ADJ-4002", notes: "Cycle count: found extra cases", actor: userManagerPP},

	// Deliberate expired-lot picks, which must raise the FR-19 audit flag.
	{kind: "pick", sku: "JUI-ORNG-1000", lot: "L-ORNG-EXPIRED", from: "PP-CENTRAL/COLD-01", quantity: "24", daysAgo: 3, reference: "SEED-PCK-2002", notes: "Deliberate expired-lot pick for the FR-19 audit flag", actor: userPickerPP},
	{kind: "pick", sku: "DRY-YOG-180", lot: "L-YOG-EXPIRED", from: "PP-CENTRAL/COLD-02", quantity: "6", daysAgo: 1, reference: "SEED-PCK-2003", notes: "Expired yoghurt withdrawn for disposal", actor: userPickerPP2},

	// Recent picks so the dashboard has fresh activity.
	{kind: "pick", sku: "COF-LATTE-240", lot: "L-LATTE-2607", from: "PP-CENTRAL/COLD-01", quantity: "24", daysAgo: 1, reference: "SEED-PCK-2004", actor: userPickerPP},
	{kind: "pick", sku: "JUI-ORNG-1000", lot: "L-ORNG-2606", from: "PP-CENTRAL/COLD-01", quantity: "18", daysAgo: 0, reference: "SEED-PCK-2005", actor: userPickerPP2},
}

// demoReservations hold stock back from the available quantity so the
// reserved and available columns are not all zero.
var demoReservations = map[ledgerKey]string{
	{location: "PP-CENTRAL/A-01-01", sku: "BEV-COLA-330", lot: "L-COLA-2601"}:   "24",
	{location: "PP-CENTRAL/COLD-01", sku: "DRY-MILK-200", lot: "L-MILK-2607"}:   "18",
	{location: "PP-CENTRAL/A-02-01", sku: "TEA-GREEN-500", lot: "L-GREEN-2609"}: "12",
}
