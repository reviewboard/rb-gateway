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


The available configuration keys are as follows:

``htpasswdPath`` (string)
    The path to the password file. Details on this can be found below.

``port`` (int)
    The port for ``rb-gateway`` to listen on. If not specified, this will
    default to 8888.

``repositories`` (array)
    The list of all repositories to host with ``rb-gateway``. See below for
    more details.

``sslCertificate`` (string)
    The path to the SSL public certificate to use when HTTPS is enabled.

``sslKey`` (string)
    The path to the SSL private key to use when HTTPS is enabled.

``tokenStorePath`` (string)
    The path to a file where ``rb-gateway`` will store authentication sessions.
    The directory for this file must exist and be writable.

``useTLS`` (boolean)
    Whether to use HTTPS for communication. This requires a valid certificate
    specified in the ``sslCertificate`` and ``sslKey`` config options.

``webhookStorePath`` (string):
    The path to a file where ``rb-gateway`` will store configured webhooks. The
    directory for this file must exist and be writable.


Each repository in the configuration file is a JSON_ object with the following
keys:

``name`` (string)
    The name to use for the repository. This is used for the configuration in
    the Review Board admin UI when linking the repository.

``path`` (string)
    The path on disk to the local repository.

``scm`` (string)
    The type of repository. This can be either ``git`` or ``hg``.


.. _JSON: https://www.json.org


Password File
=============

The password file uses the htpasswd_ format to store authentication
credentials. These credentials are used by the Review Board server to connect
to the service. This file can be created or updated with Apache's
:command:`htpasswd` tool or other widely-available third party tools.


.. _htpasswd: https://httpd.apache.org/docs/2.4/programs/htpasswd.html


.. _rb-gateway-service:

Running rb-gateway as a Service
===============================

It's likely that you'll want to run ``rb-gateway`` as a service that starts
when you boot the server. There are many ways of doing this depending on your
particular environment, but one of the more common ones is systemd_. The below
unit file example can be customized with the location of the rb-gateway binary
and your config file:

.. code-block:: ini

    [Unit]
    Description=Review Board Gateway
    After=network.target

    [Service]
    User=rb-gateway
    KillSignal=SIGTERM
    ExecStart=/usr/local/bin/rb-gateway --config /etc/rb-gateway/rb-gateway.conf
    ExecReload=/bin/kill -HUP $MAINPID

    [Install]
    WantedBy=multi-user.target


.. _systemd: https://www.freedesktop.org/wiki/Software/systemd/
