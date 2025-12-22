# About

This project is based on the Article [Scalability and Load Testing for Valorant](https://technology.riotgames.com/news/scalability-and-load-testing-valorant) by Riot Games.
The goal is to create a performant simulation of a game matchmaking system for learning purposes.

# Features

- load test harness: A self-made go service that creates player instances on parallel routines, keeps them in a pool and makes them do scenarios based on statistical distributinos like e.g. queuing for a match. Players also run multiple polling loops.
- players api service: receives player requests and updates db entries like player status.
- db: Keeps player, party and match infos
- matchmaking service: fetches players that are in queue from db and tries to create fair matches based on players mmr and the queued game mode. When a match is made it requests a provisioning of a "game server" and forwards it address to the players which then connect to it
- dind provisioning service: Provisions "Game servers" on a dind instance when matchmaking service requests it and forwards address back to it.
- All services emit metrics which get stored somewhere and queried by a grafana instance
