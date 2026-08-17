# Week 6 review checklist

## Planning and requirements

- [x] Proposal requirements reviewed and traced to the Gantt
- [x] Week 6 acceptance criteria documented for Issues #1–#6
- [x] Definition of Done documented
- [x] QA test plan and evidence format prepared
- [x] Proposal contradictions and scope clarifications recorded

## Technical foundation

- [x] Go/Fiber application starts
- [x] PostgreSQL and migrations run through Docker Compose
- [x] `/health` and `/ready` return the agreed envelope
- [x] ERD includes the required domain entities and relationships
- [x] Barcode, lot, expiry, FEFO, FIFO and audit constraints are represented
- [x] Vue, TypeScript, PrimeVue, Pinia and Router are configured
- [x] Application shell, dashboard and login routes render

## Design and barcode preparation

- [x] Core desktop and phone user journeys documented
- [x] Loading, empty, validation, error and success states specified
- [x] Responsive and accessibility requirements documented
- [x] Browser barcode libraries compared
- [x] USB/Bluetooth keyboard-wedge behavior and optional camera requirements documented
- [x] Barcode scanner test route implemented
- [ ] Real beverage barcode tested with a USB/Bluetooth hardware scanner

## Validation and handoff

- [x] Backend tests and vet pass
- [x] Frontend type-check and production build pass
- [x] Migration and Compose smoke tests pass
- [ ] Hardware scanner/device/browser evidence attached to Issue #6
- [ ] Pull request reviewed
- [ ] QA approval recorded
- [ ] Approved changes merged

The unchecked physical-device, review, QA and merge items prevent false
completion. Code-ready work moves to Code Review; only accepted and merged work
moves to Done.
