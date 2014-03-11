##Example Usage:

    $ export SG_GITHUB_TOKEN="[:token]"
    $ vCheck
    or
    $ SG_GITHUB_TOKEN="[:token]" vCheck

vCheck defaults to look in the current directory for src/. However,
you can explicitly set where vCheck should look. Examples:

    $ cd /path/to/project; vCheck
    $ vCheck /path/to/project/src
    $ vCheck -src=/path/to/project/src

##Environment Variable

vCheck uses a Personal GitHub Access Token to make http requests to GitHub.
See http://developer.github.com/v3/auth/#basic-authentication for information on
OAuth Tokens. To create a token, go here: https://github.com/settings/applications

You can set the environment variable by exporting it or setting it when calling
vCheck as shown in the example usage.
