# Freyr

This repo is a merger of three personal side projects (with various stages of progress):

- **Freyr:**
  [[original repo](https://github.com/ALeitert/freyr-rs)]
  Goal of this project was to visualise exchange rates of crypto currencies using Grafana.
  Idea was to write a Rust program which serves as [external storage for Prometheus](https://prometheus.io/docs/prometheus/latest/storage/#remote-storage-integrations).
  The protocol for that, however, has no good specification or documentation.

- **OB-Cache:**
  This project implements a data structure which stores the state of order books for a short amount of time.
  Goal is to ensure a small memory need of that data structure.
  Its repo became the current repo.
  
- **Edward:**
  This is my personal project I work on every once in a while.
  It ultimate goal would be a crypto trading bot.
  However, since I was never able to develop a working strategy, I only achieved data collection and simulation of strategies.

I also applied the structure build in my [go-template](https://github.com/ALeitert/go-template) repository.

