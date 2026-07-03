---
title: "{{title}}"
tags:
  - "{{plan|epic}}"
plan: "{{slug}}"
dateCreated: "{{date}}"
dateModified: "{{date}}"
---
## Source

(wikilink to the originating `agent-request` task, or blank if drafted in a
live chat session)

## Goal

(the CTO's original ask, lightly cleaned up)

## Tickets

(one line per child ticket — wikilink + point estimate, e.g.:
`- [[TaskNotes/Tasks/My Ticket]] — 3 pts`
Points are the Senior Planner's creation-time estimate, never edited after the
fact; drift is logged on the ticket's own Lessons Learned instead.)

## Status

(rollup counts by stage + total points vs. still-open points — e.g.:
"2 done (5 pts), 1 needs-handoff (3 pts), 1 blocked (8 pts) — 16/16 pts planned,
11 pts still open")

## Decisions Needed

(links to any child ticket currently tagged `agent-issue:bug` or
`agent-issue:needs-work`)
