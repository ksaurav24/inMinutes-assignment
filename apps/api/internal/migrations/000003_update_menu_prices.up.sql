UPDATE menu_items
SET price_paise = CASE name
    WHEN 'Classic Burger' THEN 24900
    WHEN 'Crispy Chicken Burger' THEN 28900
    WHEN 'Margherita Pizza' THEN 32900
    WHEN 'Loaded Fries' THEN 14900
    WHEN 'Caesar Salad' THEN 21900
    WHEN 'Chocolate Shake' THEN 12900
    ELSE price_paise
END,
updated_at = NOW()
WHERE name IN (
    'Classic Burger',
    'Crispy Chicken Burger',
    'Margherita Pizza',
    'Loaded Fries',
    'Caesar Salad',
    'Chocolate Shake'
);
