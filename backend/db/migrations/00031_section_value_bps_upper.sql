-- +goose Up

ALTER TABLE units ADD CONSTRAINT units_section_value_bps_upper CHECK (section_value_bps <= 10000);

-- +goose Down

ALTER TABLE units DROP CONSTRAINT IF EXISTS units_section_value_bps_upper;
