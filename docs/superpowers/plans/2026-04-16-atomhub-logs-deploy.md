# AtomHub Logs and Deployment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an authenticated request-records/usage dashboard slice to AtomHub and deploy the updated app to the user's VPS.

**Architecture:** Extend the existing request log persistence with admin read APIs and a lightweight React page for request records and token charts, while keeping the current single-binary + static-frontend architecture. Deploy by building the existing Docker Compose stack on the VPS and pointing it at the pushed `main` branch.

**Tech Stack:** Go, SQLite, React, Vite, Vitest, Docker Compose, SSH

---

### Task 1: Request records backend
- Add admin API types/handlers for recent request logs and optional model filters.
- Add store methods/tests as needed.
- Verify with targeted Go tests.

### Task 2: Request records frontend
- Add API client methods, a new authenticated page, and route/navigation entry.
- Show recent request rows and simple per-model token summaries/charts derived from backend data.
- Verify with targeted Vitest and production build.

### Task 3: End-to-end verification and docs
- Run full backend/frontend verification.
- Update README for the new page.
- Commit and push branch, then fast-forward merge into `main`.

### Task 4: VPS deployment
- Use the user-provided SSH key to connect to the VPS.
- Pull the latest `main` branch, build/restart the app via Docker Compose, and verify the app is healthy remotely.
- Report the deployed commit and access details.
