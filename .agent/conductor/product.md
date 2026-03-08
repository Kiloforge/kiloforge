# Product Definition

## Project Name

kiloforge

## Description

A local collaboration platform that orchestrates Claude Code agents for automated PR review and development workflows. A CLI tool that bridges Gitea and Claude Code to automate conductor-based development and code review.

## Problem Statement

Conductor's role-based agents (developer, reviewer) need a git forge for PRs and code review, but using GitHub adds rate limits, latency, and cost — running Gitea locally with automatic agent orchestration eliminates these friction points. There's no easy way to orchestrate multiple Claude Code agents for parallel development and automated code review without manual intervention.

## Target Users

Developers using Conductor workflows with Claude Code to orchestrate multi-agent development and review.

## Key Goals

1. **Session management** — View logs, halt agents, and resume their Claude sessions interactively
2. **Easy conductor setup and repo onboarding** — Allow users to easily set up conductor in a repo and add the repo to Gitea
