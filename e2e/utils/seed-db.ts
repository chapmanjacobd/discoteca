import * as sqlite3 from 'sqlite3';
import * as path from 'path';
import * as fs from 'fs';

export interface SeedOptions {
  databasePath?: string;
  clean?: boolean;
}

export async function seedDatabase(options: SeedOptions = {}): Promise<string> {
  const dbPath = options.databasePath || path.join(__dirname, '../fixtures/test.db');
  const shouldClean = options.clean !== false;

  // Create fixtures directory if it doesn't exist
  const fixturesDir = path.dirname(dbPath);
  if (!fs.existsSync(fixturesDir)) {
    fs.mkdirSync(fixturesDir, { recursive: true });
  }

  // Remove existing database if clean
  if (shouldClean && fs.existsSync(dbPath)) {
    fs.unlinkSync(dbPath);
  }

  return new Promise((resolve, reject) => {
    const db = new sqlite3.Database(dbPath, (err) => {
      if (err) {
        reject(new Error(`Failed to open database: ${err.message}`));
        return;
      }

      console.log('Seeding database:', dbPath);

      // Run migrations and seed data
      db.serialize(() => {
        // Create media table with full schema
        db.run(`
          CREATE TABLE IF NOT EXISTS media (
            path TEXT PRIMARY KEY,
            title TEXT,
            type TEXT,
            size INTEGER,
            duration INTEGER,
            time_created INTEGER,
            time_modified INTEGER,
            time_last_played INTEGER,
            play_count INTEGER DEFAULT 0,
            playhead REAL DEFAULT 0,
            rating INTEGER DEFAULT 0,
            transcode BOOLEAN DEFAULT 0,
            time_deleted INTEGER,
            genre TEXT,
            caption_count INTEGER DEFAULT 0,
            caption_duration INTEGER DEFAULT 0
          )
        `);

        db.run(`
          CREATE TABLE IF NOT EXISTS captions (
            rowid INTEGER PRIMARY KEY AUTOINCREMENT,
            media_path TEXT,
            time REAL,
            text TEXT
          )
        `);

        db.run(`
          CREATE TABLE IF NOT EXISTS captions_fts USING fts5 (
            text,
            content='captions',
            content_rowid='rowid'
          )
        `);

        db.run(`
          CREATE TABLE IF NOT EXISTS playlists (
            playlist_title TEXT,
            media_path TEXT,
            position INTEGER,
            PRIMARY KEY (playlist_title, media_path)
          )
        `);

        db.run(`
          CREATE TABLE IF NOT EXISTS categories (
            category TEXT,
            keyword TEXT,
            PRIMARY KEY (category, keyword)
          )
        `);

        db.run(`
          CREATE TABLE IF NOT EXISTS play_counts (
            path TEXT PRIMARY KEY,
            count INTEGER DEFAULT 0
          )
        `);

        // Insert test media after tables are created
        db.run(`INSERT OR REPLACE INTO media (path, title, type, size, duration, time_created, time_modified) VALUES
          ('/videos/movie1.mp4', 'Movie 1', 'video/mp4', 1073741824, 7200, 1704067200, 1704067200),
          ('/videos/movie2.mp4', 'Movie 2', 'video/mp4', 536870912, 5400, 1704067200, 1704067200),
          ('/videos/clip1.mp4', 'Short Clip 1', 'video/mp4', 104857600, 120, 1704067200, 1704067200),
          ('/videos/clip2.mp4', 'Short Clip 2', 'video/mp4', 52428800, 60, 1704067200, 1704067200),
          ('/audio/album/song1.mp3', 'Song 1', 'audio/mpeg', 10485760, 240, 1704067200, 1704067200),
          ('/audio/album/song2.mp3', 'Song 2', 'audio/mpeg', 8388608, 180, 1704067200, 1704067200),
          ('/audio/podcast/ep1.mp3', 'Podcast Episode 1', 'audio/mpeg', 52428800, 3600, 1704067200, 1704067200),
          ('/images/photo1.jpg', 'Photo 1', 'image/jpeg', 5242880, 0, 1704067200, 1704067200),
          ('/images/photo2.jpg', 'Photo 2', 'image/jpeg', 4194304, 0, 1704067200, 1704067200),
          ('/documents/doc1.pdf', 'Document 1', 'application/pdf', 2097152, 0, 1704067200, 1704067200)
        `);

        // Insert captions (all after 10 seconds to pass the filter)
        db.run(`INSERT INTO captions (media_path, time, text) VALUES
          ('/videos/movie1.mp4', 15.5, 'Welcome to the movie'),
          ('/videos/movie1.mp4', 30.0, 'This is an exciting scene'),
          ('/videos/movie1.mp4', 60.0, 'The plot thickens'),
          ('/videos/movie2.mp4', 20.0, 'Opening scene'),
          ('/videos/movie2.mp4', 45.0, 'Main character appears'),
          ('/videos/clip1.mp4', 12.0, 'Short clip caption'),
          ('/videos/clip2.mp4', 15.0, 'Another short clip')
        `);

        // Insert categories
        db.run(`INSERT OR REPLACE INTO categories (category, keyword) VALUES
          ('Genre', 'Action'),
          ('Genre', 'Comedy'),
          ('Genre', 'Drama'),
          ('Mood', 'Happy'),
          ('Mood', 'Sad'),
          ('Mood', 'Exciting')
        `);

        // Insert a playlist
        db.run(`INSERT OR REPLACE INTO playlists (playlist_title, media_path, position) VALUES
          ('Favorites', '/videos/movie1.mp4', 0),
          ('Favorites', '/videos/movie2.mp4', 1),
          ('Favorites', '/audio/album/song1.mp3', 2)
        `);

        db.close((err) => {
          if (err) {
            reject(new Error(`Failed to close database: ${err.message}`));
            return;
          }
          console.log('Database seeded successfully');
          resolve(dbPath);
        });
      });
    });
  });
}

export async function getDatabaseStats(dbPath: string): Promise<{
  mediaCount: number;
  captionCount: number;
  playlistCount: number;
}> {
  return new Promise((resolve, reject) => {
    const db = new sqlite3.Database(dbPath, sqlite3.OPEN_READONLY, (err) => {
      if (err) {
        reject(new Error(`Failed to open database: ${err.message}`));
        return;
      }

      const stats: any = {};

      db.get('SELECT COUNT(*) as count FROM media', (err, row: any) => {
        if (err) {
          db.close();
          reject(err);
          return;
        }
        stats.mediaCount = row.count;

        db.get('SELECT COUNT(*) as count FROM captions', (err, row: any) => {
          if (err) {
            db.close();
            reject(err);
            return;
          }
          stats.captionCount = row.count;

          db.get('SELECT COUNT(DISTINCT playlist_title) as count FROM playlists', (err, row: any) => {
            db.close();
            if (err) {
              reject(err);
              return;
            }
            stats.playlistCount = row.count;
            resolve(stats);
          });
        });
      });
    });
  });
}
