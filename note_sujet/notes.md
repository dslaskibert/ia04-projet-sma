# Simulation d’un réseau urbain de bornes de recharge pour véhicules autonomes

## Problématique

Dans un réseau urbain de bornes de recharge pour véhicules électriques autonomes, le **partage d’information entre agents** (voitures et stations) améliore-t-il réellement la **performance collective du système**, ou génère-t-il des dynamiques collectives contre-productives comme les effets de troupeau et les oscillations de charge ?

## Objectif de la simulation

On souhaite simuler un réseau de bornes de recharge pour voitures électriques autonomes dans une ville. Le but est de voir ce qu’il se passe quand les voitures ont de plus en plus d’info sur l’état du réseau pour choisir leur station.

L’intuition de base c’est de dire que plus les voitures ont d’informations, mieux elles choisissent, et mieux le réseau fonctionne. Mais en se basant sur le **paradoxe de Braess** (*l'ajout d'une nouvelle route dans un réseau routier peut réduire la performance globale, lorsque les entités se déplaçant choisissent leur route*) on part du principe inverse : **donner plus d’info peut créer des problèmes**. Si toutes les voitures voient qu’une station est vide, elles y vont toutes en même temps, et elle sature. Si l’info a 2 minutes de retard, elles prennent des décisions sur un état qui n’existe plus. On veut analyser ces phénomènes.

## Types d’agents

- **Voitures** : trajet (origine, destination), niveau de batterie, stratégie de choix de station, perception partielle du réseau.
- **Stations** : file d’attente, pompes (dont certaines en panne), diffusion de leur état, fixation dynamique de leur prix en fonction de leur taux d’occupation et du prix du courant du réseau électrique.
- **Réseau électrique** : capacité horaire variable sur la journée et prix qui en dépend.

## Stratégies de choix des voitures

- Station la **plus proche**.
- Station la **moins chère**.
- Station avec le **moins d’attente**.
- Une version combinant **les trois** critères.

## Qualités d’information disponibles

- Information **parfaite** et **instantanée**.
- Information avec **retard**.
- Information **bruitée**.
- Information **relayée** par les voitures qui se croisent.

## Indicateurs clés

- Temps d’attente moyen par voiture.
- Taux d’occupation des pompes et variance entre stations.
- Nombre de voitures en panne.
- Amplitude et période des oscillations des files (troupeau).
- Revenu moyen des stations et équité entre elles.

## Interface

Une **carte 2D** de la ville vue de dessus où l’on voit les voitures se déplacer, les files aux bornes s’allonger, les stations saturer par vagues, et les effets de troupeau apparaître en direct.

Des sliders permettront de changer le **nombre de voitures**, **de pannes**, le **prix** ou **l’offre de courant**, mais aussi la **stratégie** et la **qualité d’information** en cours de simulation pour comparer les régimes.