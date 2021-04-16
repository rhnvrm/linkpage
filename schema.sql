CREATE TABLE links (
       link_id INTEGER PRIMARY KEY,
       url TEXT NOT NULL,
       message TEXT NOT NULL,
       image_url TEXT NOT NULL,
       weight INTEGER DEFAULT 0 NOT NULL,
       hits INTEGER DEFAULT 0 NOT NULL
);
