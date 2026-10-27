# Anforderungen und Stand

Anforderungen des Auftraggebers und ihr Umsetzungsstand. Grundlage ist das Dokument „Anpassungen“
(`docs/Anpassungen.docx`, wegen Kundendaten nicht im Repository) sowie Absprachen während der Umsetzung.
Bei neuen Anforderungen diese Liste nachführen.

Legende: ✅ umgesetzt · 🟡 teilweise / bewusst anders · ⬜ offen · ⛔ bewusst nicht umgesetzt

## Anmeldung und Profil

| Anforderung | Stand | Umsetzung |
|---|---|---|
| Notfallkontakt-Beziehung als Dropdown (Vater/Mutter/Andere) | ✅ | `emergency_relation` |
| Geschlecht als Dropdown (W/M/D) | ✅ | `gender` |
| Status als Dropdown | ✅ | Pro Projekt: `schueler` (Schüler-/in aus der Region des Projekts), `gast_extern`, `mitarbeiter`. Gewählt beim Anmelden beim Projekt, Mitarbeiter vergeben nur Projekt-Admins (Teilnehmer) |
| PLZ und Ort als Dropdown, Externe mit freiem Feld | ⛔ | Freie Eingabe, Mischung aus Dropdown und Freitext nicht empfohlen |
| Freies Feld für die Klasse | ⛔ | Datenminimierung |
| Strasse und Nummer raus | 🟡 | Feld ist freiwillig |
| Apple ID als Login | ✅ | Optional, braucht ein Apple-Developer-Konto (ca. CHF 100/Jahr), siehe `docs/zugangsdaten.md` |

## Rollen, Team und Einsatz

| Anforderung | Stand | Umsetzung |
|---|---|---|
| Rollen Admin, Projektleiter, Mitarbeiter, Teilnehmer | ✅ | PL: wie MA plus Events erfassen, Bericht, Auszahlung und Zeiterfassung |
| Events erfassen und anpassen nur durch Projektleiter und Admins | ✅ | Mitarbeiter sehen Events und Teilnehmer, checken ein und erfassen Gäste |
| Funktionen mit Lohn pro Einsatz (JC 20, JSC 40, SC 80, PL über Zeiterfassung) | ✅ | Administration → Funktionen |
| Arbeitsgruppe (Team) am Event definieren, nur PL | ✅ | Event bearbeiten → Team. Der verantwortliche PL ist automatisch dabei |
| Auszahlung: alle eingeteilten Personen, Empfang mit handschriftlicher Unterschrift | ✅ | Event → Auszahlung |
| Auszahlung drucken: Name, Funktion, Betrag, Unterschrift | ✅ | Spesenblatt nach Vorlage `docs/Auszahlung.jpeg` (Papierformular von IdéeSport), mit eigenem Logo |
| Abendspesen: Name Coach, Unterschrift, Coachart mit Lohn, Total, druckbar | ✅ | Durch die Auszahlung abgedeckt |
| Spalten „Hintergrund Stunden“, „Reise Tageskurs“ der Papiervorlage | ⬜ | Keine Daten in der App, bei Bedarf als leere Spalten ergänzen |

## Events, Berichte, Zeiterfassung

| Anforderung | Stand | Umsetzung |
|---|---|---|
| Bericht zum Event: Datum, Ort, PL, Teilnehmer (W/M/D), anwesendes Team mit Funktion, Bericht, druckbar | ✅ | Event → Bericht drucken |
| Zeiterfassung: Admin bestätigt verrechnete Zahlungen mit Häkchen | ⬜ | Noch nicht umgesetzt |

## Benutzer und Notfallkontakte

| Anforderung | Stand | Umsetzung |
|---|---|---|
| Button „Notfallkontakt abrufen“ nur für berechtigte Personen | ✅ | Projekt-Admins legen Rollen und Funktionen fest, Projekt-Admins immer, Teilnehmer nie. Jeder Abruf im Audit-Protokoll |
| Gast wird als Benutzer gespeichert, gleiche Angaben, Email freiwillig | ✅ | `users.guest`, wird Mitglied des Projekts. Meldet sich der Gast später mit der Email an, übernimmt er das Konto |
| Benutzer löschen | ✅ | Nur globale Admins (Administration global → Benutzer). Check-ins bleiben anonymisiert |
| Benutzer bearbeiten, z.B. Email für Gäste ergänzen | ✅ | Nur globale Admins, protokolliert |

## Plattform

| Anforderung | Stand | Bemerkung |
|---|---|---|
| Mandantenfähigkeit (mehrere Projekte, z.B. extern/Pleno und MAAD) | 🟡 | Umgesetzt auf Branch `feat/multitenancy`, siehe unten. Datenmigration der Produktion offen |
| Hosting intern (z.B. NAS) | ⬜ | Möglich, nicht empfohlen. Aktuell Cloud Run |

## Mandantenfähigkeit (Branch `feat/multitenancy`)

Konzept und Datenmodell: [Architektur](architektur.md). Die Übernahme der bestehenden Daten wird erst nach dem Merge
geplant (Architektur → Migration der bestehenden Daten).

| Anforderung | Stand | Umsetzung |
|---|---|---|
| Landingpage ohne Projekt: Login mit Google oder Apple, danach Projekt aus Dropdown wählen | ✅ | Startseite mit den eigenen Projekten, Projektwahl auch in der Kopfzeile |
| Neue Benutzer bzw. neues Projekt: Menü „Nach Projekten suchen“ mit Volltextsuche | ✅ | Suche in Name, Region, Beschreibung und Kurzname, jedes Wort muss vorkommen |
| Midnightsports Aadorf ist das erste Projekt | ✅ | Im Schema angelegt (Kurzname `aadorf`) mit Funktionen, Orten und Notfallzugriff |
| Globale Administration, in der Projekte erfasst werden | ✅ | Administration global → Alle Projekte (nur `ADMIN_EMAIL`) |
| Beim Erfassen: Name, Logo, Projekt-Admin, danach alles wie bei Aadorf | ✅ | Onboarding mit Funktionen, Standardort, Notfallzugriff (Projektleiter) und Projekt-Admin per Email |
| Logo hochladen, Standard ist das bestehende Logo | ✅ | PNG, JPEG oder WebP bis 512 KB (SVG aus Sicherheitsgründen nicht), änderbar unter Alle Projekte |
| Seite Benutzer heisst neu Teilnehmer | ✅ | Im Projekt: Teilnehmer (nur Mitglieder dieses Projekts) |
| Teilnehmer sind global, in der globalen Administration unter Benutzer | ✅ | Personen und Profil global, Liste mit allen Projekten pro Person |
| URL-Parameter führt beim Scan des QR-Codes direkt zum Projekt | ✅ | `/?projekt=<kurzname>`, Link und QR-Code unter Administration des Projekts. Auch über den Login hinweg |
| Bezeichnungen JC/JSC/SC/PL beim Erstellen festlegen | ✅ | Onboarding: Bezeichnungen und Löhne frei, weitere Funktionen möglich |
| Projekt-Admin sieht nur Benutzer seines Projekts | ✅ | Teilnehmerliste, Suche und alle Zugriffe gefiltert nach Projekt |
| Anmeldung beim Projekt nur über Landingpage oder Projekt-Link | ✅ | Ohne Mitgliedschaft kein Zugriff, kein Check-in (Meldung beim Scannen) |
| Manuelle Änderungen an Benutzern nur durch globale Admins, nachgeführt im Audit | ✅ | Profil ändern und löschen nur global, Audit-Protokoll unter Administration global → Audit |
| Gesetzlich relevante Vorgänge (Daten Minderjähriger, CH/EU) im Audit | 🟡 | Zugriffe auf Notfallkontakte und Profile, Änderungen, Löschungen, Exporte, Rollen, Verwarnungen, Teilen. Rechtliche Prüfung, Aufbewahrungsdauer und Einwilligung der Eltern noch offen |
| Teilnehmer verwarnen oder sperren | ✅ | Teilnehmer → Stift (Projektleiter, Projekt-Admins), mit Grund |
| Meldung beim Scannen und beim manuellen Check-in | ✅ | Verwarnt: eingecheckt mit gelber Meldung. Gesperrt: nicht eingecheckt, rote Meldung, keine Event-Anmeldung |
| Frist: entsperren, auf verwarnt zurücksetzen oder Verwarnung löschen | ✅ | Administration des Projekts → Verwarnungen und Sperren |
| Verwarnungen nur im auslösenden Projekt sichtbar | ✅ | Ändern nur im auslösenden Projekt |
| Teilen zwischen zwei Organisationen, beim Erstellen festgelegt, blockiert bis zur Einwilligung | ✅ | Onboarding → teilen mit. Neues Projekt ausstehend bis der Partner-Admin einwilligt (Administration → Geteilte Verwarnungen), Stand in der Liste der Projekte |
| Datenmigration der Produktion | ⬜ | Nach erfolgreichem Pitch und Merge |
