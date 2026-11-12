select id, email, password_hash, created_at
from users
where users.email = $1;