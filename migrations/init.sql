CREATE TABLE IF NOT EXISTS users (
                                     id SERIAL PRIMARY KEY,
                                     vendor_id INT NOT NULL
);

CREATE TABLE IF NOT EXISTS tokens (
                                      user_id INT REFERENCES users(id),
                                      token VARCHAR(255) NOT NULL,
                                      validated BOOLEAN NOT NULL,
                                      PRIMARY KEY (user_id, token)
);
