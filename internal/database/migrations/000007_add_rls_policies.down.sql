-- Drop policies and disable RLS for 'categories'
DROP POLICY IF EXISTS "Allow modification for authenticated users" ON categories;
DROP POLICY IF EXISTS "Allow public select access" ON categories;
ALTER TABLE categories DISABLE ROW LEVEL SECURITY;

-- Drop policies and disable RLS for 'products'
DROP POLICY IF EXISTS "Allow modification for authenticated users" ON products;
DROP POLICY IF EXISTS "Allow public select access" ON products;
ALTER TABLE products DISABLE ROW LEVEL SECURITY;

-- Drop policies and disable RLS for 'order_items'
DROP POLICY IF EXISTS "Allow select based on order owner" ON order_items;
ALTER TABLE order_items DISABLE ROW LEVEL SECURITY;

-- Drop policies and disable RLS for 'orders'
DROP POLICY IF EXISTS "Allow insert for authenticated users" ON orders;
DROP POLICY IF EXISTS "Allow select access to owner" ON orders;
ALTER TABLE orders DISABLE ROW LEVEL SECURITY;

-- Drop policies and disable RLS for 'cart_items'
DROP POLICY IF EXISTS "Allow access based on cart owner" ON cart_items;
ALTER TABLE cart_items DISABLE ROW LEVEL SECURITY;

-- Drop policies and disable RLS for 'carts'
DROP POLICY IF EXISTS "Allow full access to owner" ON carts;
ALTER TABLE carts DISABLE ROW LEVEL SECURITY;

-- Drop policies and disable RLS for 'addresses'
DROP POLICY IF EXISTS "Allow full access to owner" ON addresses;
ALTER TABLE addresses DISABLE ROW LEVEL SECURITY;

-- Drop policies and disable RLS for 'users'
DROP POLICY IF EXISTS "Allow individual update access" ON users;
DROP POLICY IF EXISTS "Allow individual select access" ON users;
ALTER TABLE users DISABLE ROW LEVEL SECURITY;