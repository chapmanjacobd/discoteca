import * as sqlite3 from 'sqlite3';
import * as path from 'path';
import * as fs from 'fs';

export interface SeedOptions {
  databasePath?: string;
  clean?: boolean;
}

export async function seedDatabase(options: SeedOptions = {}): Promise<string> {
  const dbPath = options.databasePath || path.join(__dirname, '../../e2e/fixtures/test.db');
  const shouldClean = options.clean !== false;

  // Create fixtures directory if it doesn't exist
  const fixturesDir = path.dirname(dbPath);
  if (!fs.existsSync(fixturesDir)) {
    fs.mkdirSync(fixturesDir, { recursive: true });
  }

  // Remove existing database if clean
  if (shouldClean && fs.existsSync(dbPath)) {
    fs.unlinkSync(dbPath);
    // Also remove WAL and SHM files if they exist
    try { fs.unlinkSync(dbPath + '-wal'); } catch {}
    try { fs.unlinkSync(dbPath + '-shm'); } catch {}
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
        // Enable foreign keys and WAL mode
        db.run('PRAGMA foreign_keys = ON');
        db.run('PRAGMA journal_mode = WAL');

        // Use the actual schema from internal/commands/schema.sql
        // (Simplified for seeding purposes but keeping correct structure and triggers)

        db.run(`CREATE TABLE IF NOT EXISTS media (
            path TEXT PRIMARY KEY,
            title TEXT,
            duration INTEGER,
            size INTEGER,
            time_created INTEGER,
            time_modified INTEGER,
            time_deleted INTEGER DEFAULT 0,
            time_first_played INTEGER DEFAULT 0,
            time_last_played INTEGER DEFAULT 0,
            play_count INTEGER DEFAULT 0,
            playhead INTEGER DEFAULT 0,
            type TEXT,
            width INTEGER,
            height INTEGER,
            fps REAL,
            video_codecs TEXT,
            audio_codecs TEXT,
            subtitle_codecs TEXT,
            video_count INTEGER DEFAULT 0,
            audio_count INTEGER DEFAULT 0,
            subtitle_count INTEGER DEFAULT 0,
            album TEXT,
            artist TEXT,
            genre TEXT,
            mood TEXT,
            bpm INTEGER,
            key TEXT,
            decade TEXT,
            categories TEXT,
            city TEXT,
            country TEXT,
            description TEXT,
            language TEXT,
            webpath TEXT,
            uploader TEXT,
            time_uploaded INTEGER,
            time_downloaded INTEGER,
            view_count INTEGER,
            num_comments INTEGER,
            favorite_count INTEGER,
            score REAL,
            upvote_ratio REAL,
            latitude REAL,
            longitude REAL
        )`);

        db.run(`CREATE TABLE IF NOT EXISTS captions (
            rowid INTEGER PRIMARY KEY AUTOINCREMENT,
            media_path TEXT NOT NULL,
            time REAL,
            text TEXT,
            FOREIGN KEY (media_path) REFERENCES media(path) ON DELETE CASCADE
        )`);

        db.run(`CREATE VIRTUAL TABLE IF NOT EXISTS captions_fts USING fts5(
            media_path UNINDEXED,
            text,
            content='captions',
            content_rowid='rowid'
        )`);

        db.run(`CREATE TRIGGER IF NOT EXISTS captions_ai AFTER INSERT ON captions BEGIN
            INSERT INTO captions_fts(rowid, media_path, text)
            VALUES (new.rowid, new.media_path, new.text);
        END;`);

        db.run(`CREATE TABLE IF NOT EXISTS playlists (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            path TEXT UNIQUE,
            title TEXT,
            extractor_key TEXT,
            extractor_config TEXT,
            time_deleted INTEGER DEFAULT 0
        )`);

        db.run(`CREATE TABLE IF NOT EXISTS playlist_items (
            playlist_id INTEGER NOT NULL,
            media_path TEXT NOT NULL,
            track_number INTEGER,
            time_added INTEGER DEFAULT (strftime('%s', 'now')),
            PRIMARY KEY (playlist_id, media_path),
            FOREIGN KEY (playlist_id) REFERENCES playlists(id) ON DELETE CASCADE,
            FOREIGN KEY (media_path) REFERENCES media(path) ON DELETE CASCADE
        )`);

        db.run(`CREATE TABLE IF NOT EXISTS history (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            media_path TEXT NOT NULL,
            time_played INTEGER DEFAULT (strftime('%s', 'now')),
            playhead INTEGER,
            done INTEGER,
            FOREIGN KEY (media_path) REFERENCES media(path) ON DELETE CASCADE
        )`);

        db.run(`CREATE TABLE IF NOT EXISTS custom_keywords (
            category TEXT NOT NULL,
            keyword TEXT NOT NULL,
            PRIMARY KEY (category, keyword)
        )`);

        db.run(`CREATE VIRTUAL TABLE IF NOT EXISTS media_fts USING fts5(
            path,
            title,
            content='media',
            content_rowid='rowid'
        )`);

        db.run(`CREATE TRIGGER IF NOT EXISTS media_ai AFTER INSERT ON media BEGIN
            INSERT INTO media_fts(rowid, path, title)
            VALUES (new.rowid, new.path, new.title);
        END;`);

        // Insert test media
        db.run(`INSERT OR REPLACE INTO media (path, title, type, size, duration, time_created, time_modified, score) VALUES
          ('/videos/movie1.mp4', 'Movie 1', 'video', 1073741824, 7200, 1704067200, 1704067200, 5),
          ('/videos/movie2.mp4', 'Movie 2', 'video', 536870912, 5400, 1704067200, 1704067200, 4),
          ('/videos/clip1.mp4', 'Short Clip 1', 'video', 104857600, 120, 1704067200, 1704067200, 3),
          ('/videos/clip2.mp4', 'Short Clip 2', 'video', 52428800, 60, 1704067200, 1704067200, 2),
          ('/audio/album/song1.mp3', 'Song 1', 'audio', 10485760, 240, 1704067200, 1704067200, 5),
          ('/audio/album/song2.mp3', 'Song 2', 'audio', 8388608, 180, 1704067200, 1704067200, 4),
          ('/audio/podcast/ep1.mp3', 'Podcast Episode 1', 'audio', 52428800, 3600, 1704067200, 1704067200, 3),
          ('/images/photo1.jpg', 'Photo 1', 'image', 5242880, 0, 1704067200, 1704067200, 0),
          ('/images/photo2.jpg', 'Photo 2', 'image', 4194304, 0, 1704067200, 1704067200, 0),
          ('/documents/doc1.pdf', 'Document 1', 'application/pdf', 2097152, 0, 1704067200, 1704067200, 0)
        `);

        // Insert captions
        db.run(`INSERT INTO captions (media_path, time, text) VALUES
          ('/videos/movie1.mp4', 15.5, 'Welcome to the movie'),
          ('/videos/movie1.mp4', 30.0, 'This is an exciting scene'),
          ('/videos/movie1.mp4', 60.0, 'The plot thickens'),
          ('/videos/movie2.mp4', 20.0, 'Opening scene'),
          ('/videos/movie2.mp4', 45.0, 'Main character appears'),
          ('/videos/clip1.mp4', 12.0, 'Short clip caption'),
          ('/videos/clip2.mp4', 15.0, 'Another short clip')
        `);

        // Insert a playlist
        db.run(`INSERT OR REPLACE INTO playlists (id, path, title) VALUES
          (1, 'fav-playlist', 'Favorites')
        `);

        db.run(`INSERT OR REPLACE INTO playlist_items (playlist_id, media_path, track_number) VALUES
          (1, '/videos/movie1.mp4', 0),
          (1, '/videos/movie2.mp4', 1),
          (1, '/audio/album/song1.mp3', 2)
        `);

        db.close((err) => {
          if (err) {
            reject(new Error(`Failed to close database: ${err.message}`));
            return;
          }
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

          db.get('SELECT COUNT(DISTINCT title) as count FROM playlists', (err, row: any) => {
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
