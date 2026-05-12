.. _rb-gateway-running:

==================
Running rb-gateway
==================

Once ``rb-gateway`` is :ref:`installed <rb-gateway-installation>` and
:ref:`configured <rb-gateway-configuration>`, start the server with the
``serve`` command:

.. code-block:: console

    $ ./rb-gateway serve

By default, this reads ``config.json`` from the current directory. Use
``--config`` to specify a different path:

.. code-block:: console

    $ ./rb-gateway --config /etc/rb-gateway/config.json serve

The server reloads its configuration automatically when the config file
changes or when it receives ``SIGHUP``. Send ``SIGINT`` or ``SIGTERM`` to
shut down gracefully.


Commands
========

``rb-gateway`` supports the following commands:

``serve``
    Start the API server. This is the default if no command is given.

``trigger-webhooks <repository> <event>``
    Trigger matching webhooks for a repository and event.

``reinstall-hooks``
    Re-install hook scripts for all repositories.

The version can be printed with ``--version``:

.. code-block:: console

    $ ./rb-gateway --version


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
    ExecStart=/usr/local/bin/rb-gateway --config /etc/rb-gateway/config.json serve
    ExecReload=/bin/kill -HUP $MAINPID

    [Install]
    WantedBy=multi-user.target


.. _systemd: https://www.freedesktop.org/wiki/Software/systemd/
