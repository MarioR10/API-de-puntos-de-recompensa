
insert into users(email,password_hash
) values ($1,$2)
    RETURNING id, email, password_hash, created_at