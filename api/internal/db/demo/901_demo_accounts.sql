-- Erstelle Account für Louis Kern --

INSERT INTO users (email, first_name, last_name, birth_date, gender, street, zip, city,
                   emergency_name, emergency_phone, emergency_relation)
VALUES ('louis.baumann93@gmail.com', 'Hodor', 'Hodor', '1993-09-06', 'm', 'Unterdorfstrasse 78', '9507', 'Kranenberg',
        'Tschumbolina Kern', '077 789 25 18', 'andere')
ON CONFLICT (email) DO UPDATE SET
    first_name         = COALESCE(NULLIF(users.first_name, ''), EXCLUDED.first_name),
    last_name          = COALESCE(NULLIF(users.last_name, ''), EXCLUDED.last_name),
    birth_date         = COALESCE(users.birth_date, EXCLUDED.birth_date),
    gender             = COALESCE(NULLIF(users.gender, ''), EXCLUDED.gender),
    street             = COALESCE(NULLIF(users.street, ''), EXCLUDED.street),
    zip                = COALESCE(NULLIF(users.zip, ''), EXCLUDED.zip),
    city               = COALESCE(NULLIF(users.city, ''), EXCLUDED.city),
    emergency_name     = COALESCE(NULLIF(users.emergency_name, ''), EXCLUDED.emergency_name),
    emergency_phone    = COALESCE(NULLIF(users.emergency_phone, ''), EXCLUDED.emergency_phone),
    emergency_relation = COALESCE(NULLIF(users.emergency_relation, ''), EXCLUDED.emergency_relation);

-- Projekt-Admin in Aadorf (Admins sind Mitarbeiter mit Funktion)
INSERT INTO project_members (project_id, user_id, role, status, function_id)
SELECT p.id, u.id, 'admin', 'mitarbeiter', f.id
FROM projects p
JOIN functions f ON f.project_id = p.id AND f.name = 'PL'
CROSS JOIN users u
WHERE p.slug = 'aadorf' AND u.email = 'louis.baumann93@gmail.com'
ON CONFLICT (project_id, user_id) DO NOTHING;

-- Erstelle Account für Roman Hofmeister --

INSERT INTO users (email, first_name, last_name, birth_date, gender, street, zip, city,
                   emergency_name, emergency_phone, emergency_relation)
VALUES ('roman.hofi@gmail.com', 'Roman', 'Hofmeister', '1999-10-06', 'm', 'Hauptstrasse 22', '8357', 'Gunti',
        'Stefan Hofmeister', '078 584 52 96', 'vater')
ON CONFLICT (email) DO UPDATE SET
    first_name         = COALESCE(NULLIF(users.first_name, ''), EXCLUDED.first_name),
    last_name          = COALESCE(NULLIF(users.last_name, ''), EXCLUDED.last_name),
    birth_date         = COALESCE(users.birth_date, EXCLUDED.birth_date),
    gender             = COALESCE(NULLIF(users.gender, ''), EXCLUDED.gender),
    street             = COALESCE(NULLIF(users.street, ''), EXCLUDED.street),
    zip                = COALESCE(NULLIF(users.zip, ''), EXCLUDED.zip),
    city               = COALESCE(NULLIF(users.city, ''), EXCLUDED.city),
    emergency_name     = COALESCE(NULLIF(users.emergency_name, ''), EXCLUDED.emergency_name),
    emergency_phone    = COALESCE(NULLIF(users.emergency_phone, ''), EXCLUDED.emergency_phone),
    emergency_relation = COALESCE(NULLIF(users.emergency_relation, ''), EXCLUDED.emergency_relation);

-- Projekt-Admin in Aadorf (Admins sind Mitarbeiter mit Funktion)
INSERT INTO project_members (project_id, user_id, role, status, function_id)
SELECT p.id, u.id, 'admin', 'mitarbeiter', f.id
FROM projects p
JOIN functions f ON f.project_id = p.id AND f.name = 'PL'
CROSS JOIN users u
WHERE p.slug = 'aadorf' AND u.email = 'roman.hofi@gmail.com'
ON CONFLICT (project_id, user_id) DO NOTHING;