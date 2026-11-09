-- Demo-Daten, nur für die Demo (nie in Produktion). Vor dem Build nach api/internal/db/migrations/ kopieren,
-- danach wieder entfernen (siehe docs/deployment-demo.md).
--
-- 4 Projekte mit je 350 Mitgliedern: Midnightsports Aadorf (aus 001_init.sql) und drei neue.
-- Pro Projekt sind 175 Mitglieder nur dort, 175 zusätzlich bei einem Nachbarprojekt (Ring: 1-2, 2-3, 3-4, 4-1).
-- Total 1050 Personen. Pro Projekt ein Projekt-Admin, 3 Projektleiter und 8 Mitarbeiter mit Funktion.
-- Pro Projekt jeden Freitag ein Event: 6 vergangene (Check-ins, Bericht) und kommende bis Ende des nächsten Jahres.
-- Team und Anmeldungen für die vergangenen und die nächsten 6 Events.
-- Alle Personen sind erfundene Profile mit Email @example.org (ohne Login).

INSERT INTO projects (slug, name, region, description) VALUES
    ('pleno-frauenfeld', 'Pleno Frauenfeld', 'Frauenfeld', 'Jugendtreff mit offenen Sportabenden'),
    ('jugendtreff-muenchwilen', 'Jugendtreff Münchwilen', 'Münchwilen', 'Offener Jugendtreff und Midnight Sport'),
    ('midnightsports-wil', 'Midnightsports Wil', 'Wil', 'Offene Sportnächte in Wil');

-- Reihenfolge der Projekte im Ring, mit PLZ und Ort für die Adressen der Mitglieder
CREATE TEMP TABLE demo_projects ON COMMIT DROP AS
SELECT s.idx, p.id, s.zip, s.city
FROM (VALUES (0, 'aadorf', '8355', 'Aadorf'),
             (1, 'pleno-frauenfeld', '8500', 'Frauenfeld'),
             (2, 'jugendtreff-muenchwilen', '9542', 'Münchwilen'),
             (3, 'midnightsports-wil', '9500', 'Wil')) AS s (idx, slug, zip, city)
JOIN projects p ON p.slug = s.slug;

-- Neue Projekte: Funktionen (Pleno mit eigenen Bezeichnungen), Notfallzugriff für Projektleiter, Standardort
INSERT INTO functions (project_id, name, salary)
SELECT p.id, f.name, f.salary
FROM projects p
JOIN (VALUES ('pleno-frauenfeld', 'Teamleitung', NULL::numeric), ('pleno-frauenfeld', 'Helfer/in', 25),
             ('pleno-frauenfeld', 'Coach', 50),
             ('jugendtreff-muenchwilen', 'JC', 20), ('jugendtreff-muenchwilen', 'JSC', 40),
             ('jugendtreff-muenchwilen', 'SC', 80), ('jugendtreff-muenchwilen', 'PL', NULL),
             ('midnightsports-wil', 'JC', 20), ('midnightsports-wil', 'JSC', 40),
             ('midnightsports-wil', 'SC', 80), ('midnightsports-wil', 'PL', NULL)) AS f (slug, name, salary)
  ON f.slug = p.slug;

INSERT INTO emergency_access_roles (project_id, role)
SELECT id, 'projektleiter' FROM projects WHERE slug IN ('pleno-frauenfeld', 'jugendtreff-muenchwilen', 'midnightsports-wil');

INSERT INTO locations (project_id, name, street, zip, city, is_default)
SELECT p.id, l.name, l.street, l.zip, l.city, true
FROM projects p
JOIN (VALUES ('pleno-frauenfeld', 'Jugendtreff Pleno', 'Rheinstrasse 12', '8500', 'Frauenfeld'),
             ('jugendtreff-muenchwilen', 'Turnhalle Münchwilen', 'Schulstrasse 5', '9542', 'Münchwilen'),
             ('midnightsports-wil', 'Sporthalle Lindenhof', 'Lindenhofstrasse 20', '9500', 'Wil'))
  AS l (slug, name, street, zip, city)
  ON l.slug = p.slug;

-- Personen: n 1-700 gehören zu je einem Projekt (175 pro Projekt), n 701-1050 zu zwei benachbarten Projekten.
-- Die geteilten Gruppen sind abwechselnd 88 und 87 gross, so hat jedes Projekt 175 + 88 + 87 = 350 Mitglieder.
CREATE TEMP TABLE demo_people ON COMMIT DROP AS
SELECT n, (n - 1) / 175 AS home, NULL::int AS partner, (n - 1) % 175 + 1 AS pos, gen_random_uuid() AS user_id
FROM generate_series(1, 700) AS n
UNION ALL
SELECT 701 + k, g, (g + 1) % 4, NULL, gen_random_uuid()
FROM generate_series(0, 349) AS k
CROSS JOIN LATERAL (SELECT CASE WHEN k < 88 THEN 0 WHEN k < 175 THEN 1 WHEN k < 263 THEN 2 ELSE 3 END AS g) AS grp;

INSERT INTO users (id, email, first_name, last_name, birth_date, gender, street, zip, city,
                   emergency_name, emergency_phone, emergency_relation)
SELECT d.user_id,
       translate(lower(f.name || '.' || l.name || '.' || d.n), 'äöüéè', 'aouee') || '@example.org',
       f.name, l.name,
       -- 12 bis 20 Jahre alt
       date '2014-10-01' - ((d.n * 97) % (365 * 8)),
       g.gender,
       (ARRAY['Dorfstrasse', 'Bahnhofstrasse', 'Schulstrasse', 'Kirchgasse', 'Hauptstrasse'])[d.n % 5 + 1] || ' ' || (d.n % 60 + 1),
       p.zip, p.city,
       (ARRAY['Andrea', 'Daniel', 'Sandra', 'Thomas', 'Monika', 'Peter'])[d.n % 6 + 1] || ' ' || l.name,
       '079 ' || lpad(((d.n * 7919) % 1000000)::text, 6, '0'),
       (ARRAY['mutter', 'vater', 'andere'])[d.n % 3 + 1]
FROM demo_people d
JOIN demo_projects p ON p.idx = d.home
CROSS JOIN LATERAL (SELECT CASE WHEN d.n % 41 = 0 THEN 'd' WHEN d.n % 2 = 0 THEN 'w' ELSE 'm' END AS gender) AS g
CROSS JOIN LATERAL (SELECT CASE WHEN g.gender = 'w'
    THEN (ARRAY['Lea', 'Mia', 'Laura', 'Sara', 'Lena', 'Anna', 'Nina', 'Elena', 'Jana', 'Lara', 'Chiara', 'Alina'])[d.n % 12 + 1]
    ELSE (ARRAY['Noah', 'Luca', 'Leon', 'Elias', 'Jan', 'Nico', 'Levin', 'Robin', 'Tim', 'Samuel', 'Jonas', 'Dario'])[d.n % 12 + 1]
    END AS name) AS f
CROSS JOIN LATERAL (SELECT (ARRAY['Müller', 'Meier', 'Schmid', 'Keller', 'Weber', 'Huber', 'Schneider', 'Steiner',
    'Brunner', 'Baumann', 'Gerber', 'Frei', 'Fischer', 'Moser', 'Graf', 'Wyss', 'Zimmermann', 'Bachmann', 'Kälin',
    'Bühler'])[(d.n * 7) % 20 + 1] AS name) AS l;

-- Mitgliedschaften im eigenen Projekt: die ersten 12 Personen jedes Projekts sind das Team
-- (1 Projekt-Admin, 3 Projektleiter, 8 Mitarbeiter), jede fünfte Person ist Gast extern
INSERT INTO project_members (project_id, user_id, role, status)
SELECT p.id, d.user_id,
       CASE WHEN d.pos = 1 THEN 'admin' WHEN d.pos <= 4 THEN 'projektleiter' WHEN d.pos <= 12 THEN 'mitarbeiter'
            ELSE 'teilnehmer' END,
       CASE WHEN d.pos <= 12 THEN 'mitarbeiter' WHEN d.n % 5 = 0 THEN 'gast_extern' ELSE 'schueler' END
FROM demo_people d
JOIN demo_projects p ON p.idx = d.home;

-- Mitgliedschaft im Nachbarprojekt: dort Gast extern
INSERT INTO project_members (project_id, user_id, role, status)
SELECT p.id, d.user_id, 'teilnehmer', 'gast_extern'
FROM demo_people d
JOIN demo_projects p ON p.idx = d.partner;

-- Funktionen des Teams: Admin und Projektleiter die Funktion ohne Lohn, Mitarbeiter reihum die übrigen
UPDATE project_members pm
SET function_id = CASE
    WHEN pm.role IN ('admin', 'projektleiter') THEN
        (SELECT f.id FROM functions f WHERE f.project_id = pm.project_id AND f.salary IS NULL ORDER BY f.id LIMIT 1)
    ELSE
        (SELECT f.id FROM functions f WHERE f.project_id = pm.project_id AND f.salary IS NOT NULL
         ORDER BY f.id OFFSET (d.pos % (SELECT count(*) FROM functions f2
                                       WHERE f2.project_id = pm.project_id AND f2.salary IS NOT NULL)) LIMIT 1)
    END
FROM demo_people d
WHERE d.user_id = pm.user_id AND pm.role <> 'teilnehmer';

-- Events relativ zum Tag des Starts (die Demo-Datenbank wird bei jedem Start neu angelegt): pro Projekt die letzten
-- 6 Freitage und alle Freitage bis Ende des nächsten Jahres, 21:00 bis 24:00 Uhr. Ort reihum, verantwortlich reihum
-- ein Projektleiter. Der dritte kommende Event ist abgesagt.
CREATE TEMP TABLE demo_events ON COMMIT DROP AS
SELECT p.id AS project_id, w AS week,
       ((date_trunc('week', current_date)::date + 4 + 7 * w) + time '21:00') AT TIME ZONE 'Europe/Zurich' AS starts_at
FROM demo_projects p
CROSS JOIN generate_series(-6, 110) AS w
WHERE w <> 0
  AND date_trunc('week', current_date)::date + 4 + 7 * w < date_trunc('year', current_date)::date + interval '2 years';

INSERT INTO events (project_id, starts_at, ends_at, location_id, leader_id, cancelled, created_by)
SELECT e.project_id, e.starts_at, e.starts_at + interval '3 hours',
       (SELECT l.id FROM locations l WHERE l.project_id = e.project_id
        ORDER BY l.id OFFSET abs(e.week) % (SELECT count(*) FROM locations l2 WHERE l2.project_id = e.project_id) LIMIT 1),
       pl.user_id, e.week = 3, pl.user_id
FROM demo_events e
CROSS JOIN LATERAL (
    SELECT m.user_id FROM project_members m
    WHERE m.project_id = e.project_id AND m.role = 'projektleiter'
    ORDER BY m.user_id OFFSET abs(e.week) % 3 LIMIT 1) AS pl;

-- Weiter in der Zukunft sind Events noch nicht geplant: Team und Anmeldungen nur bis in 6 Wochen
-- Team pro Event: der verantwortliche Projektleiter und 3 Mitarbeiter, Funktion und Lohn wie beim Einteilen
INSERT INTO event_staff (event_id, user_id, function_id, salary)
SELECT e.id, m.user_id, m.function_id, f.salary
FROM events e
JOIN project_members m ON m.project_id = e.project_id
JOIN functions f ON f.id = m.function_id
WHERE e.starts_at < now() + interval '6 weeks'
  AND (m.user_id = e.leader_id
   OR m.user_id IN (SELECT x.user_id FROM project_members x
                    WHERE x.project_id = e.project_id AND x.role = 'mitarbeiter'
                    ORDER BY md5(x.user_id::text || e.id) LIMIT 3));

-- Anmeldungen: etwa 15 % der Mitglieder pro Event (Teilnehmer und Gäste beider Status)
INSERT INTO registrations (event_id, user_id, created_at)
SELECT e.id, m.user_id, e.starts_at - interval '3 days'
FROM events e
JOIN project_members m ON m.project_id = e.project_id AND m.role = 'teilnehmer'
WHERE e.starts_at < now() + interval '6 weeks'
  AND abs(hashtext(m.user_id::text || e.id)) % 100 < 15;

-- Check-ins an vergangenen Events: drei Viertel der Angemeldeten und einige ohne Anmeldung
INSERT INTO checkins (event_id, user_id, checked_in_at, checked_in_by)
SELECT e.id, m.user_id, e.starts_at + ((abs(hashtext(m.user_id::text)) % 90) * interval '1 minute'), e.leader_id
FROM events e
JOIN project_members m ON m.project_id = e.project_id AND m.role = 'teilnehmer'
LEFT JOIN registrations r ON r.event_id = e.id AND r.user_id = m.user_id
WHERE e.ends_at < now()
  AND ((r.user_id IS NOT NULL AND abs(hashtext(m.user_id::text || e.id || 'c')) % 4 <> 0)
    OR (r.user_id IS NULL AND abs(hashtext(m.user_id::text || e.id || 'w')) % 60 = 0));

-- Bericht zu jedem vergangenen Event
INSERT INTO reports (event_id, body, author_id, updated_at)
SELECT e.id,
       (ARRAY['Ruhiger Abend, viel Fussball und Unihockey. Keine Vorkommnisse.',
              'Volle Halle, Basketball-Turnier am Schluss. Ein Ball ist kaputt gegangen.',
              'Weniger Teilnehmende wegen Schulferien. Gute Stimmung.',
              'Neue Gesichter dabei, zwei Gäste aus der Nachbargemeinde. Material vollständig.'])[abs(hashtext(e.id::text)) % 4 + 1],
       e.leader_id, e.ends_at + interval '1 hour'
FROM events e
WHERE e.ends_at < now();
