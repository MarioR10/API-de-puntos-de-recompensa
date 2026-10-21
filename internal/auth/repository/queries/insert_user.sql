
insert into users(id,email,password_hash
) values ($1,$2, $3)
    RETURNING id, email, password_hash, created_at