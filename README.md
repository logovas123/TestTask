# Тестовое задание
## API for online library of songs

Проект предоставляет api для взаимодествие с библотекой. Для хранения данных используется PostgreSQL. 

Сгенерированный swagger с подробным описанием api лежит в папке docs. Ниже кратко описаны rest-методы.

1. GET /api/songs - возвращает список песен
2. POST /api/songs -добавление новой песни
3. DELETE /api/song/{SONG_ID} - удаление песни по id
4. PUT /api/song/{SONG_ID} - обновление данных песни по id
5. GET /api/song/{SONG_ID} - получение текста песни по id
   
Для запуска проекта:

1) Склонируйте репозиторий:
```
git clone git@github.com:logovas123/TestTask.git
```
2) Создайте и заполните .env файл, согласно файлу env-example
3) API ходит во внешний сервис, так что его тоже необходимо поднять и прописать его хост и порт в env файле
4) Поднимите проект и базу:
```
docker compose up --build -d
```
6) Complete!
