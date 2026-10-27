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
| Status als Dropdown | ✅ | `schueler_aadorf`, `gast_extern`, `mitarbeiter`. Mitarbeiter vergeben nur Admins (Benutzer → Stift) |
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
| Auszahlung abschliessen: Zusammenfassung, Warnung bei fehlenden Unterschriften, Unterschrift des zuständigen PL | ✅ | Auszahlung → Abschliessen. Danach sind die Unterschriften gesperrt, der Abschluss kann aufgehoben werden |
| Auszahlung drucken: Name, Funktion, Betrag, Unterschrift | ✅ | Erst nach dem Abschluss (im Abschlussfenster). | Spesenblatt nach Vorlage `docs/Auszahlung.jpeg` (Papierformular von IdéeSport), mit eigenem Logo |
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
| Button „Notfallkontakt abrufen“ nur für berechtigte Personen | ✅ | Admins legen Rollen und Funktionen fest, Admins immer, Teilnehmer nie |
| Gast wird als Benutzer gespeichert, gleiche Angaben, Email freiwillig | ✅ | `users.guest`. Meldet sich der Gast später mit der Email an, übernimmt er das Konto |
| Admins können Benutzer löschen | ✅ | Check-ins bleiben anonymisiert |
| Admins können Benutzer bearbeiten, z.B. Email für Gäste ergänzen | ✅ | Benutzer → Stift |

## Plattform

| Anforderung | Stand | Bemerkung |
|---|---|---|
| Mandantenfähigkeit (mehrere Projekte, z.B. extern/Pleno und MAAD) | ⬜ | Grösserer Umbau: Berechtigungskonzept, evtl. Identity Provider (z.B. Zitadel), ca. 2–4 Tage |
| Hosting intern (z.B. NAS) | ⬜ | Möglich, nicht empfohlen. Aktuell Cloud Run |
