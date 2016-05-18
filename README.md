# honeybadger-s3
A simple Go command line utility to save Honeybadger.io data to AWS S3

## Overview
`honeybadger-s3` was created to save honeybadger.io data in AWS S3, for the purpose of backup, and opportunity for later analysis. `honeybadger-s3` can pull data from one or more projects, it can do this incrementally or pull everything every time it runs.

## Docker
You can easily run this out of a docker container. This project comes with a Dockerfile and ./build.sh script to create your docker image. Inside the docker container this project makes use of the (docker-cron)[https://github.com/MasteryConnect/docker-cron] project. `docker-cron` allows easy configuration in docker of a cron process that also keeps the docker container up and running. The ./build.sh script builds a linux binary, located at ./bin/honeybadger-s3.

## Examples
After running ./build.sh:

Run the linux binary
```
./bin/honeybadger-s3 --help
```

Run in a docker container passing in the necessary environment variables to configure `honeybadger-s3` and `docker-cron`
```
docker run -v ~/.aws/credentials:/root/.aws/credentials --name=honeybadger-s3 -e "DC_SECS=*/5" -e "S3_BUCKET=mc-metrics" -e "PROJECTS=mindful" -e "S3_DIRECTORY=honeybadger" masteryconnect/honeybadger-s3
```

Run just honeybadger-s3 (without docker-cron) from the docker container.
```
docker run -it --rm -v ~/.aws/credentials:/root/.aws/credentials masteryconnect/honeybadger-s3:1.0 honeybadger-s3 --help
```

To see the help text
```
honeybadger-s3 --help

NAME:
   honeybadger-s3 -
   backup honeybadger.io faults to AWS S3.

   For S3 access credentials, one of the following is required:
   1. set up the following environment variables:
   AWS_ACCESS_KEY_ID or AWS_ACCESS_KEY
   AWS_SECRET_ACCESS_KEY or AWS_SECRET_KEY
   2. set up ~/.aws/credentials (shared credentials)
   3. run from an ec2 machine and user that has permission to S3 (ec2 role)


USAGE:
   honeybadger-s3 [global options] command [command options] [arguments...]

VERSION:
   1.0

COMMANDS:
   help, h      Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --s3-bucket, -b              AWS S3 bucket to backup to [$S3_BUCKET]
   --s3-directory, -d           (optional) the directory in the AWS S3 bucket to back up to [$S3_DIRECTORY]
   --projects, -p               (optional) comma separated list of projects to backup. If not set, all projects are backed up [$PROJECTS]
   --honeybadger-key, -k        your Honeybadger.io API key [$HB_API_KEY]
   --last-run, -l               the last time this process ran, the time from which this will search for new faults. Use the following format: <year><month><day><hour><minute><second> e.g. 20150430140508 [$LAST_RUN]
   --help, -h                   show help
   --version, -v                print the version
```

## License

The MIT License (MIT)

Copyright (c) 2016 MasteryConnect

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
