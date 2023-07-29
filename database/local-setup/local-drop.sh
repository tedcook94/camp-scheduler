#!/bin/bash

psql postgres://postgres:postgres@localhost:5432?sslmode=disable -f local-drop.sql
