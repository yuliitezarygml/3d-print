INSERT INTO users (name, email, password_hash, role)
VALUES ('Administrator', 'admin@printforge.local', crypt('admin12345', gen_salt('bf', 12)), 'ADMIN')
ON CONFLICT (email) DO NOTHING;

INSERT INTO printers (name, manufacturer, model, status, build_x_mm, build_y_mm, build_z_mm, power_watts, purchase_price)
VALUES
  ('X1 Carbon', 'Bambu Lab', 'X1 Carbon', 'PRINTING', 256, 256, 256, 350, 34900),
  ('P1S', 'Bambu Lab', 'P1S', 'IDLE', 256, 256, 256, 300, 23900),
  ('MK4', 'Prusa', 'MK4', 'IDLE', 250, 210, 220, 120, 24900),
  ('K1 Max', 'Creality', 'K1 Max', 'MAINTENANCE', 300, 300, 300, 350, 18900)
ON CONFLICT DO NOTHING;

INSERT INTO filament_spools (code, manufacturer, product_name, material, color_name, color_hex, initial_weight_grams, remaining_weight_grams, purchase_price, supplier)
VALUES
  ('SP-0001', 'Bambu Lab', 'PLA Basic', 'PLA', 'Black', '#151515', 1000, 623, 450, 'Bambu Lab'),
  ('SP-0002', 'Bambu Lab', 'PLA Basic', 'PLA', 'White', '#F4F4F0', 1000, 890, 450, 'Bambu Lab'),
  ('SP-0003', 'eSUN', 'PETG', 'PETG', 'Black', '#202124', 1000, 126, 430, 'Local supplier'),
  ('SP-0004', 'Polymaker', 'PolyLite ASA', 'ASA', 'Grey', '#777B80', 1000, 760, 610, 'Polymaker'),
  ('SP-0005', 'eSUN', 'eTPU-95A', 'TPU', 'Black', '#101010', 1000, 540, 590, 'Local supplier')
ON CONFLICT (code) DO NOTHING;

INSERT INTO customers (name, company, phone, email) VALUES
  ('Andrei Rusu', 'Rusu Engineering', '+373 69 000 101', 'andrei@example.com'),
  ('Elena Popa', 'Atelier Forma', '+373 68 000 202', 'elena@example.com')
ON CONFLICT DO NOTHING;

INSERT INTO orders (number, customer_id, status, selling_price, paid_amount, deadline, notes)
SELECT 'ORD-2026-00001', id, 'PRINTING', 790, 400, now() + interval '2 days', 'Gear housing, 12 pcs'
FROM customers WHERE email = 'andrei@example.com'
ON CONFLICT (number) DO NOTHING;

