.. _rb-gateway-configuration:

================================
Configuring Review Board Gateway
================================

``rb-gateway`` is configured with two files: a configuration file and a
password file.


Configuration File
==================

The configuration file is a JSON_ file which defines the basic settings for the
service, and includes an array listing each of the local repositories which
``rb-gateway`` should allow access to:

.. code-block:: javascript

    {
        "htpasswdPath": "/etc/rb-gateway/htpasswd",
        "port": 8888,
        "tokenStorePath": "/var/lib/rb-gateway/tokens.dat",
        "webhookStorePath": "/etc/rb-gateway/webhooks.json",
        "repositories": [
            {"name": "repo1", "path": "/path/to/repo1.git", "scm": "git"},
            {"name": "repo2", "path": "/path/to/repo2.hg", "scm": "hg"}
        ]
    }


Relative paths in the configuration file are resolved relative to the
directory containing the configuration file.

The available configuration keys are as follows:

``htpasswdPath`` (string)
    The path to the :ref:`password file <rb-gateway-htpasswd-file>`.

    Defaults to :file:`htpasswd`.

``port`` (int)
    The port for ``rb-gateway`` to listen on.

    Defaults to ``8888``.

``repositories`` (array of object)
    The list of all repositories to host with ``rb-gateway``. This key is
    required.

    Each repository in the configuration file is a JSON_ object with the following
    keys:

    ``name`` (string)
        The name to use for the repository. This is used for the configuration in
        the Review Board admin UI when linking the repository, and appears in API
        URLs.

    ``path`` (string)
        The path on disk to the local repository.

    ``scm`` (string)
        The type of repository. This can be either ``git`` or ``hg``.

``sslCertificate`` (string)
    The path to the SSL public certificate to use when HTTPS is enabled.

    Required if ``useTLS`` is ``true``.

``sslKey`` (string)
    The path to the SSL private key to use when HTTPS is enabled.

    Required if ``useTLS`` is ``true``.

``tokenStorePath`` (string)
    The path to a file where ``rb-gateway`` will store authentication sessions.
    The directory for this file must exist and be writable.

    Defaults to ``tokens.dat``.

``useTLS`` (boolean)
    Whether to use HTTPS for communication. This requires a valid certificate
    specified in the ``sslCertificate`` and ``sslKey`` config options.

    Defaults to ``false``.

``webhookStorePath`` (string)
    The path to the :ref:`WebHook configuration file
    <rb-gateway-webhooks-configuration>`.

    Defaults to :file:`webhooks.json`.


.. _JSON: https://www.json.org


.. _rb-gateway-htpasswd-file:

Password File
=============

The password file uses the htpasswd_ format to store authentication
credentials. These credentials are used by the Review Board server to connect
to the service. This file can be created or updated with Apache's
:command:`htpasswd` tool or other widely-available third party tools.

Passwords may be stored as bcrypt hashes (recommended) or in plain text. For
example, to create a new htpasswd file with a bcrypt-hashed password:

.. code-block:: console

    $ htpasswd -Bc /etc/rb-gateway/htpasswd myuser


.. _htpasswd: https://httpd.apache.org/docs/2.4/programs/htpasswd.html


.. _rb-gateway-webhooks-configuration:

WebHooks File
=============

The WebHook configuration file is a JSON_ file containing a list of
configurations for WebHooks which should be emitted when new code is pushed.

The available configuration keys are as follows. All keys are required.

``enabled`` (boolean)
    Whether the WebHook is enabled.

``events`` (array of string)
    The set of events to notify on. This currently only supports ``"push"``.

``id`` (string)
    A unique ID for the WebHook.

``repos`` (array of string)
    The set of repositories to trigger this WebHook configuration for.

``secret`` (string)
    A secret to use for generating an HMAC-SHA1 signature for the payload.

``url`` (string)
    The URL to dispatch the WebHook to.


Example
-------

.. code-block:: javascript

    [
        {
            "enabled": true,
            "events": ["push"],
            "id": "repo1-reviewboard",
            "repos": ["repo1"],
            "secret": "<secret from Review Board>",
            "url": "https://reviewboard.example.com/repos/1/rbgateway/hooks/close-submitted/"
        }
    ]
