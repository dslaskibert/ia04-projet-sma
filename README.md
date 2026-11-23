# Simulation d’un réseau urbain de bornes de recharge pour véhicules autonomes

Projet de groupe en **IA04 (Systèmes Multi-Agents)** à l'UTC, en automne 2026

Simulation multi-agent en Go d'un réseau de bornes de recharge pour véhicules électriques autonomes, visant à étudier l'impact du partage d'information entre agents sur la performance collective du réseau. [Note détaillant le sujet](notes_sujet/notes.md)

## Groupe

- David Slaski Bert
- Iago Rhuda Ramos
- Judith Lajouanie
- Arthur Erzinger De Lima

Suivi par M. Sylvain Lagrue

## Structure du projet

```
agent/ interfaces et implémentations des agents (voitures, stations, ...)
environment/ graphes et cartes
simulation/ moteur de simulation
types/ types partagés entre packages
cmd/simulation/ fichier main
frontend/ interface web React
backend/ API HTTP Go
```

## Lancement du projet

### Simulation

```
go run cmd/simulation/main.go
```

### Backend

```sh
cd backend
go run .
```

Le serveur écoute par défaut sur `http://localhost:8080`. La variable
d'environnement `PORT` permet de modifier le port.

### Frontend

```sh
cd frontend
npm install
npm run dev
```

En développement, Vite transmet les requêtes `/api` au backend sur le port
`8080`.
