UPDATE menu_items
SET price_paise = CASE name
    WHEN 'Classic Burger' THEN 899
    WHEN 'Crispy Chicken Burger' THEN 999
    WHEN 'Margherita Pizza' THEN 1099
    WHEN 'Loaded Fries' THEN 499
    WHEN 'Caesar Salad' THEN 699
    WHEN 'Chocolate Shake' THEN 399
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
