docker swarm init
chown 1000:1000 -R shuffle-database/
docker network create -d overlay shuffle_prod
docker stack deploy --compose-file=orborus.yml shuffle_orborus
