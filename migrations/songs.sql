DROP TABLE IF EXISTS songs;
CREATE TABLE songs (
    song_id SERIAL PRIMARY KEY,
    song_name varchar(100) NOT NULL,
    group_name varchar(100) NOT NULL,
    release_date varchar(50) NOT NULL,
    text_of_song TEXT NOT NULL,
    link varchar(300) NOT NULL
);