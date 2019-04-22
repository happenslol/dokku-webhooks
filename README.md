# Dokku webhooks plugin

**This project is a work-in-progress and not usable yet!**

This dokku plugin will enable you to trigger dokku commands through webhooks you define, secured by secrets you set. [bolt](https://github.com/boltdb/bolt) is used as a backing storage to keep track of secrets and hooks. This is how the plugin basically works:

* An app called `webhooks-server` is created and started. You can add domains to this server so that you can reach it under your defined address.
* A storage with 2 unix sockets is created and mounted to the `webhooks-server`:
    * [dokku daemon](https://github.com/dokku/dokku-daemon), which is used to send commands from the container to dokku running on the host,
    * another socket used for communication between the dokku cli and the `webhooks-server`.
* The server then creates its databases inside the mounted folder:
    * The hooks database, containing secrets and the hooks for each app,
    * the jobs database, where entries will be pushed when a webhook is triggered and progress for running jobs will be tracked.
* Webhooks can be added, removed and configured through the cli and will be passed to the server through the socket.
* When the server receives a `POST` request, it will check the body against the secrets database and then look for the hook's ID. It will then look for any variables in the command and replace them with given query parameters, and output the command the the daemon's socket.

Documentation and usage instructions will be added when basic functionality is working.
