-- +goose Up
INSERT INTO package_types (code,max_weight,extra_price) VALUES
  ('bag',10,5), ('box',30,20),
  ('film',0,1), ('bag+film',10,6), ('box+film',30,21);

-- +goose Down
DELETE FROM package_types
WHERE code IN ('bag','box','film','bag+film','box+film');