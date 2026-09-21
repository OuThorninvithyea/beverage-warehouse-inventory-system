package main

// Demo fixtures for local development and manual QA. Everything here is
// fictional: the warehouses, the beverages and the GS1 prefixes are chosen so
// the dataset can never collide with a real trade item.
//
// Product barcodes use the 884 (Cambodia) EAN-13 prefix plus two UPC-A values
// for imported goods, so both supported formats are exercised. Location
// barcodes use the 20x restricted-circulation prefix, which is reserved for
// internal use. Every check digit is valid; demo_test.go enforces that against
// the same validator the API uses.

type demoWarehouse struct {
	code    string
	name    string
	address string
}

type demoLocation struct {
	warehouse string
	code      string
	zone      string
	aisle     string
	rack      string
	shelf     string
	barcode   string
	pickable  bool
}

type demoUser struct {
	email     string
	fullName  string
	role      string
	warehouse string // empty means the user is not scoped to one warehouse
}

type demoCategory struct {
	name   string
	parent string // empty for a root category
}

type demoProduct struct {
	sku        string
	name       string
	category   string
	unit       string
	barcode    string
	lotTracked bool
}

type demoLot struct {
	sku             string
	number          string
	expiresInDays   int // negative is already expired
	hasExpiry       bool
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
	warehousePhnomPenh = "PP-CENTRAL"
	warehouseSiemReap  = "SR-DEPOT"

	userManager   = "manager@bwims.local"
	userPicker    = "picker@bwims.local"
	userPickerSR  = "picker.sr@bwims.local"
	userViewer    = "viewer@bwims.local"
	demoReference = "SEED-"
)

var demoWarehouses = []demoWarehouse{
	{code: warehousePhnomPenh, name: "Phnom Penh Central DC", address: "Street 271, Sen Sok, Phnom Penh"},
	{code: warehouseSiemReap, name: "Siem Reap Depot", address: "National Road 6, Siem Reap"},
}

var demoLocations = []demoLocation{
	{warehouse: warehousePhnomPenh, code: "RECV-DOCK", zone: "RECEIVING", barcode: "2001000190010", pickable: false},
	{warehouse: warehousePhnomPenh, code: "QUAR-01", zone: "QUARANTINE", barcode: "2001000190027", pickable: false},
	{warehouse: warehousePhnomPenh, code: "A-01-01", zone: "AMBIENT", aisle: "A", rack: "01", shelf: "01", barcode: "2001000101016", pickable: true},
	{warehouse: warehousePhnomPenh, code: "A-01-02", zone: "AMBIENT", aisle: "A", rack: "01", shelf: "02", barcode: "2001000101023", pickable: true},
	{warehouse: warehousePhnomPenh, code: "COLD-01", zone: "CHILLED", aisle: "C", rack: "01", shelf: "01", barcode: "2001000102013", pickable: true},
	{warehouse: warehouseSiemReap, code: "RECV-DOCK", zone: "RECEIVING", barcode: "2002000190017", pickable: false},
	{warehouse: warehouseSiemReap, code: "A-01-01", zone: "AMBIENT", aisle: "A", rack: "01", shelf: "01", barcode: "2002000101013", pickable: true},
	{warehouse: warehouseSiemReap, code: "A-01-02", zone: "AMBIENT", aisle: "A", rack: "01", shelf: "02", barcode: "2002000101020", pickable: true},
}

var demoUsers = []demoUser{
	{email: userManager, fullName: "Demo Warehouse Manager", role: "warehouse_manager", warehouse: warehousePhnomPenh},
	{email: userPicker, fullName: "Demo Picker Phnom Penh", role: "picker", warehouse: warehousePhnomPenh},
	{email: userPickerSR, fullName: "Demo Picker Siem Reap", role: "picker", warehouse: warehouseSiemReap},
	{email: userViewer, fullName: "Demo Viewer", role: "viewer"},
}

var demoCategories = []demoCategory{
	{name: "Beverages"},
	{name: "Carbonated Soft Drinks", parent: "Beverages"},
	{name: "Water", parent: "Beverages"},
	{name: "Juice", parent: "Beverages"},
	{name: "Energy Drinks", parent: "Beverages"},
	{name: "Beer", parent: "Beverages"},
	{name: "Dairy and Coffee", parent: "Beverages"},
	{name: "Packaging"},
}

var demoProducts = []demoProduct{
	{sku: "BEV-COLA-330", name: "Angkor Cola Can 330 ml", category: "Carbonated Soft Drinks", unit: "case", barcode: "8841000001015", lotTracked: true},
	{sku: "BEV-COLA-1500", name: "Angkor Cola PET 1.5 L", category: "Carbonated Soft Drinks", unit: "case", barcode: "8841000001022", lotTracked: true},
	{sku: "BEV-LEMON-330", name: "Mekong Lemon Soda Can 330 ml", category: "Carbonated Soft Drinks", unit: "case", barcode: "8841000001039", lotTracked: true},
	{sku: "WTR-STILL-500", name: "Kulen Still Water 500 ml", category: "Water", unit: "case", barcode: "8841000002012", lotTracked: true},
	{sku: "WTR-SPARK-750", name: "Kulen Sparkling Water 750 ml", category: "Water", unit: "case", barcode: "8841000002029", lotTracked: true},
	{sku: "JUI-ORNG-1000", name: "Tonle Orange Juice 1 L", category: "Juice", unit: "case", barcode: "8841000003019", lotTracked: true},
	{sku: "JUI-MANGO-250", name: "Tonle Mango Nectar 250 ml", category: "Juice", unit: "case", barcode: "8841000003026", lotTracked: true},
	{sku: "ENG-BOOST-250", name: "Bayon Boost Energy 250 ml", category: "Energy Drinks", unit: "case", barcode: "8841000004016", lotTracked: true},
	{sku: "BEE-LAGER-330", name: "Riverside Lager Can 330 ml", category: "Beer", unit: "case", barcode: "8841000005013", lotTracked: true},
	{sku: "BEE-STOUT-330", name: "Riverside Stout Can 330 ml", category: "Beer", unit: "case", barcode: "8841000005020", lotTracked: true},
	{sku: "COF-LATTE-240", name: "Phnom Coffee Latte Can 240 ml", category: "Dairy and Coffee", unit: "case", barcode: "8841000006010", lotTracked: true},
	{sku: "DRY-MILK-200", name: "Siem Dairy Milk 200 ml", category: "Dairy and Coffee", unit: "case", barcode: "8841000006027", lotTracked: true},
	{sku: "IMP-COLA-355", name: "Imported Cola Can 355 ml", category: "Carbonated Soft Drinks", unit: "case", barcode: "049000012347", lotTracked: true},
	{sku: "IMP-ROOT-355", name: "Imported Root Beer Can 355 ml", category: "Carbonated Soft Drinks", unit: "case", barcode: "012000012341", lotTracked: true},
	{sku: "PKG-CRATE-24", name: "Returnable Crate 24 Slot", category: "Packaging", unit: "crate", barcode: "8841000007017", lotTracked: false},
}

// Expiry dates are relative to the seed run so FEFO, near-expiry alerting and
// the expired-lot audit flag stay demonstrable no matter when the data is
// loaded.
var demoLots = []demoLot{
	{sku: "BEV-COLA-330", number: "L-COLA-2512", expiresInDays: 60, hasExpiry: true, receivedDaysAgo: 120},
	{sku: "BEV-COLA-330", number: "L-COLA-2601", expiresInDays: 180, hasExpiry: true, receivedDaysAgo: 40},
	{sku: "BEV-COLA-1500", number: "L-COLA15-2602", expiresInDays: 150, hasExpiry: true, receivedDaysAgo: 50},
	{sku: "BEV-LEMON-330", number: "L-LEMON-2604", expiresInDays: 210, hasExpiry: true, receivedDaysAgo: 33},
	{sku: "WTR-STILL-500", number: "L-WTR-2701", expiresInDays: 540, hasExpiry: true, receivedDaysAgo: 60},
	{sku: "WTR-SPARK-750", number: "L-SPARK-2705", expiresInDays: 500, hasExpiry: true, receivedDaysAgo: 70},
	{sku: "JUI-ORNG-1000", number: "L-ORNG-EXPIRED", expiresInDays: -5, hasExpiry: true, receivedDaysAgo: 95},
	{sku: "JUI-ORNG-1000", number: "L-ORNG-2606", expiresInDays: 21, hasExpiry: true, receivedDaysAgo: 10},
	{sku: "JUI-MANGO-250", number: "L-MANGO-2602", expiresInDays: 100, hasExpiry: true, receivedDaysAgo: 35},
	{sku: "ENG-BOOST-250", number: "L-BOOST-2605", expiresInDays: 300, hasExpiry: true, receivedDaysAgo: 25},
	{sku: "BEE-LAGER-330", number: "L-LAGER-2603", expiresInDays: 240, hasExpiry: true, receivedDaysAgo: 30},
	{sku: "BEE-STOUT-330", number: "L-STOUT-2606", expiresInDays: 200, hasExpiry: true, receivedDaysAgo: 28},
	{sku: "COF-LATTE-240", number: "L-LATTE-2607", expiresInDays: 12, hasExpiry: true, receivedDaysAgo: 20},
	{sku: "DRY-MILK-200", number: "L-MILK-2607", expiresInDays: 7, hasExpiry: true, receivedDaysAgo: 14},
	{sku: "IMP-COLA-355", number: "L-IMP-2604", expiresInDays: 270, hasExpiry: true, receivedDaysAgo: 45},
	{sku: "IMP-ROOT-355", number: "L-ROOT-2603", expiresInDays: 260, hasExpiry: true, receivedDaysAgo: 42},
}

// demoMovements is ordered oldest first for readability; the ledger sorts by
// daysAgo anyway.
var demoMovements = []demoMovement{
	{kind: "receive", sku: "BEV-COLA-330", lot: "L-COLA-2512", to: "PP-CENTRAL/A-01-01", quantity: "240", unitCost: "9.20", daysAgo: 120, reference: "SEED-RCV-1001", actor: userManager},
	{kind: "receive", sku: "JUI-ORNG-1000", lot: "L-ORNG-EXPIRED", to: "PP-CENTRAL/COLD-01", quantity: "120", unitCost: "14.50", daysAgo: 95, reference: "SEED-RCV-1002", actor: userManager},
	{kind: "receive", sku: "WTR-SPARK-750", lot: "L-SPARK-2705", to: "PP-CENTRAL/A-01-02", quantity: "180", unitCost: "7.80", daysAgo: 70, reference: "SEED-RCV-1003", actor: userManager},
	{kind: "receive", sku: "WTR-STILL-500", lot: "L-WTR-2701", to: "PP-CENTRAL/A-01-02", quantity: "400", unitCost: "4.10", daysAgo: 60, reference: "SEED-RCV-1004", actor: userManager},
	{kind: "receive", sku: "BEV-COLA-1500", lot: "L-COLA15-2602", to: "PP-CENTRAL/A-01-01", quantity: "150", unitCost: "12.40", daysAgo: 50, reference: "SEED-RCV-1005", actor: userManager},
	{kind: "receive", sku: "IMP-COLA-355", lot: "L-IMP-2604", to: "PP-CENTRAL/A-01-02", quantity: "96", unitCost: "18.90", daysAgo: 45, reference: "SEED-RCV-1006", actor: userManager},
	{kind: "receive", sku: "IMP-ROOT-355", lot: "L-ROOT-2603", to: "SR-DEPOT/A-01-01", quantity: "60", unitCost: "19.75", daysAgo: 42, reference: "SEED-RCV-1007", actor: userManager},
	{kind: "receive", sku: "BEV-COLA-330", lot: "L-COLA-2601", to: "PP-CENTRAL/A-01-01", quantity: "300", unitCost: "9.65", daysAgo: 40, reference: "SEED-RCV-1008", actor: userManager},
	{kind: "receive", sku: "JUI-MANGO-250", lot: "L-MANGO-2602", to: "PP-CENTRAL/A-01-02", quantity: "200", unitCost: "10.30", daysAgo: 35, reference: "SEED-RCV-1009", actor: userManager},
	{kind: "receive", sku: "BEV-LEMON-330", lot: "L-LEMON-2604", to: "PP-CENTRAL/A-01-01", quantity: "180", unitCost: "9.05", daysAgo: 33, reference: "SEED-RCV-1010", actor: userManager},
	{kind: "receive", sku: "BEE-LAGER-330", lot: "L-LAGER-2603", to: "PP-CENTRAL/A-01-01", quantity: "264", unitCost: "21.40", daysAgo: 30, reference: "SEED-RCV-1011", actor: userManager},
	{kind: "receive", sku: "BEE-STOUT-330", lot: "L-STOUT-2606", to: "SR-DEPOT/A-01-01", quantity: "120", unitCost: "23.10", daysAgo: 28, reference: "SEED-RCV-1012", actor: userManager},
	{kind: "receive", sku: "ENG-BOOST-250", lot: "L-BOOST-2605", to: "PP-CENTRAL/COLD-01", quantity: "144", unitCost: "16.75", daysAgo: 25, reference: "SEED-RCV-1013", actor: userManager},
	{kind: "receive", sku: "COF-LATTE-240", lot: "L-LATTE-2607", to: "PP-CENTRAL/COLD-01", quantity: "96", unitCost: "15.20", daysAgo: 20, reference: "SEED-RCV-1014", actor: userManager},
	{kind: "receive", sku: "PKG-CRATE-24", to: "PP-CENTRAL/RECV-DOCK", quantity: "500", unitCost: "2.25", daysAgo: 18, reference: "SEED-RCV-1015", notes: "Returnable crates, not lot tracked", actor: userManager},
	{kind: "receive", sku: "DRY-MILK-200", lot: "L-MILK-2607", to: "PP-CENTRAL/COLD-01", quantity: "240", unitCost: "11.40", daysAgo: 14, reference: "SEED-RCV-1016", actor: userManager},
	{kind: "pick", sku: "BEV-COLA-330", lot: "L-COLA-2512", from: "PP-CENTRAL/A-01-01", quantity: "60", daysAgo: 12, reference: "SEED-PCK-2001", notes: "FEFO drew the earliest expiring cola lot", actor: userPicker},
	{kind: "receive", sku: "JUI-ORNG-1000", lot: "L-ORNG-2606", to: "PP-CENTRAL/COLD-01", quantity: "180", unitCost: "14.95", daysAgo: 10, reference: "SEED-RCV-1017", actor: userManager},
	{kind: "pick", sku: "BEE-LAGER-330", lot: "L-LAGER-2603", from: "PP-CENTRAL/A-01-01", quantity: "48", daysAgo: 9, reference: "SEED-PCK-2002", actor: userPicker},
	{kind: "transfer", sku: "BEV-COLA-330", lot: "L-COLA-2601", from: "PP-CENTRAL/A-01-01", to: "SR-DEPOT/A-01-01", quantity: "120", daysAgo: 7, reference: "SEED-TRF-3001", notes: "Interwarehouse replenishment", actor: userManager},
	{kind: "transfer", sku: "PKG-CRATE-24", from: "PP-CENTRAL/RECV-DOCK", to: "PP-CENTRAL/A-01-02", quantity: "200", daysAgo: 6, reference: "SEED-TRF-3002", notes: "Putaway from the receiving dock", actor: userManager},
	{kind: "pick", sku: "WTR-STILL-500", lot: "L-WTR-2701", from: "PP-CENTRAL/A-01-02", quantity: "150", daysAgo: 5, reference: "SEED-PCK-2003", actor: userPicker},
	{kind: "adjust", sku: "BEE-STOUT-330", lot: "L-STOUT-2606", from: "SR-DEPOT/A-01-01", quantity: "6", daysAgo: 4, reference: "SEED-ADJ-4001", notes: "Cycle count: damaged in handling", actor: userPickerSR},
	{kind: "pick", sku: "JUI-ORNG-1000", lot: "L-ORNG-EXPIRED", from: "PP-CENTRAL/COLD-01", quantity: "24", daysAgo: 3, reference: "SEED-PCK-2004", notes: "Deliberate expired-lot pick for the FR-19 audit flag", actor: userPicker},
	{kind: "adjust", sku: "WTR-STILL-500", lot: "L-WTR-2701", to: "PP-CENTRAL/A-01-02", quantity: "12", unitCost: "4.10", daysAgo: 2, reference: "SEED-ADJ-4002", notes: "Cycle count: found extra cases", actor: userManager},
	{kind: "pick", sku: "DRY-MILK-200", lot: "L-MILK-2607", from: "PP-CENTRAL/COLD-01", quantity: "36", daysAgo: 1, reference: "SEED-PCK-2005", actor: userPicker},
	{kind: "pick", sku: "COF-LATTE-240", lot: "L-LATTE-2607", from: "PP-CENTRAL/COLD-01", quantity: "24", daysAgo: 1, reference: "SEED-PCK-2006", actor: userPicker},
}

// demoReservations hold stock back from the available quantity so the UI has
// something to show in the reserved column.
var demoReservations = map[ledgerKey]string{
	{location: "PP-CENTRAL/A-01-01", sku: "BEV-COLA-330", lot: "L-COLA-2601"}:   "24",
	{location: "PP-CENTRAL/COLD-01", sku: "ENG-BOOST-250", lot: "L-BOOST-2605"}: "12",
}
