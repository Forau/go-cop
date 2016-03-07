# go-cop
GO - Command Parser
===================

This is a library that tries to make it conveient to add commands to your go-program.
It is dependant on the excilent [liner](https://github.com/peterh/liner) library, for handling the tty, history, emacs-keys and more.

What gocop adds on top is a small parser to sugest autocomplete and invoke commands easily.
History/autosugest is content aware, so default arguments with the same name will share sugestions, but you can also create custom sugestions. For example, email addresses might be sugested from an address book, or reply might be sugested from messages recived.

Status
------

Verion 0.0 [![Build Status](https://travis-ci.org/Forau/go-cop.svg?branch=master)](https://travis-ci.org/Forau/go-cop)
Still in very early stage. Everything might change.

Usage
-----

The basic usage will be to create a 'world' struct, and hook commands, and arguments onto it.
Arguments can be optional or greedy.

Arguments are separated by whitespace, so if you need to send an argument with spaces or tabs, you need to eighter backslash the whitespace, or put the string in single or double quotes.

Bugs / Todo
-----------

Linebreak during quoted strings does not allow you to continue on next line
Greedy arguments are not yet implemented
Make easier to use
Create examples
Make 'worlds' swappable, so commands can control following states.

and more...





