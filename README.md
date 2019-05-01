# dokku webhooks plugin

**This project is a work-in-progress and not usable yet!**

dokku-webhooks lets you issue custom dokku commands by triggering webhooks. The core functionality is implemented and working, but the glue that holds everything together is not all there yet. The following things still need to happen before this plugin is usable in production:

- [ ] Implement log recording for executed commands
- [ ] Add an install script
- [ ] Implement `listen` and `stop` commands
- [ ] Improve overall logging quality
- [ ] Add tests for server and cli commands
- [ ] Add documentation for installation and usage

The workflow will look like this:

* Install the plugin using `dokku plugin:install https://github.com/happenslol/dokku-webhooks.git`
* Enable the webhook server: `dokku webhooks:listen`
* Generate a secret for your app: `dokku webhooks:gen-secret <app> --length 64`
* Enable webhooks for your app: `dokku webhooks:enable <app>`
* Create a webhook. The command you pass can contain variables that will be substituted with url query params you pass. Additionally, there's available params like the app name, which enable you to easily write commands.

```bash
# A post request to /foo/webhook1 with the secret as the body will trigger the command ps:rebuild foo
dokku webhooks:create foo webhook1 "ps:rebuild #app"

# Posting the secret to /foo/webhook2?cmd=stop will run ps:stop foo
dokku webhooks:create foo webhook2 "ps:#cmd #app"
```

* If you want to manually trigger a webhook to test if it works, you can run `dokku webhooks:trigger foo webhook2 --args "cmd=stop"`
* Using `dokku webhooks:logs foo webhook2`, you can see the most recent output of the command
