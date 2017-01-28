## Docker volume driver for MinFS.

# Install instructions.
- Fetch and build the driver.
  ```sh
  $ go get github.com/minio/minfs-docker-volume
  ```
- Run the Driver.
  ```
  $ $GOPATH/bin/minfs-docker-volume --mountroot=/mnt/minfs/
  ```
- Create a volume using the driver. Pass Minio server info as options shown below.
  ```
  $  $ docker volume create -d minfs \
     --name medical-imaging-store \
     -o endpoint=https://play.minio.io:9000 \
     -o access_key=Q3AM3UQ867SPQQA43P2F \
     -o secret-key=zuf+tfteSlswRu7BJ86wekitnifILbZam1KYY3TG \ 
     -o bucket=test-bucket
  ```
  
 - Share the new volume with a container and start using it.
   ```
   docker run -it -v medical-imaging-store:/data busybox /bin/sh
   ```
 
