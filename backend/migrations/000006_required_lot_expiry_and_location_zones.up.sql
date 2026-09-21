ALTER TABLE lots
    ADD CONSTRAINT lots_expiration_required CHECK (expiration_date IS NOT NULL);

ALTER TABLE locations
    ADD CONSTRAINT locations_zone_valid CHECK (
        zone IS NULL OR UPPER(zone) IN ('AMBIENT', 'CHILLED', 'FROZEN', 'RECEIVING', 'QUARANTINE')
    );
