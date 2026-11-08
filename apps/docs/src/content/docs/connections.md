---
title: Connections
description: What a connection is, where to get a key, what a paid connection costs, what leaves your computer, and the address of a local server.
---

A **connection** is a place Numen can send requests to for a plugin, and the way in: a service you pay for, a server you run yourself, or an agent program on your computer. You enter it once, on the **Connections** page of the settings, and the plugins that need a model use it. The model is chosen where the connection is used, so one key can serve several plugins with different models.

## Adding a connection

Open **Settings**, choose **Connections**, and press **Add connection**. Choose the type, give the connection a name if you want one, and fill in what the type asks for. Leave the name empty and Numen names it after the type, such as "OpenRouter", and "OpenRouter 2" for the next one. The types come from your plugins; a type whose plugin is not installed is listed as **Type unavailable** and is skipped.

## Where to get a key

A service gives you a key in your account on its website, usually under a heading such as API keys. Copy the key, paste it into the **API key** field of the Add dialog, and save. Numen keeps the key in the system keychain and never shows it again. To replace it, edit the connection and type the new one. To remove it, press **Remove key** and save.

The key belongs to the address of the connection. If you change the address, Numen removes the stored key and asks for it again, so a key is never sent to a server you did not enter it for.

## What a paid connection costs

The service charges your account for the requests Numen sends, by the size of the text and the model you choose. Numen does not know the price and does not limit it. Check the price list of the service before you pick a large model, and set a spending limit in your account where the service offers one. An agent connection, such as one for a program that runs on your computer, uses the subscription or login of that program.

## What leaves your computer

A request sends the text a plugin asks the model about, such as a note being proofread or the text being indexed for search, to the address of the connection, with the key. Nothing is sent until a plugin uses the connection. A key is sent over a plain `http` address only when the address is on your own computer.

## A server on your computer or network

A server you run yourself, such as Ollama, listens on an address like `http://localhost:11434/v1`. Choose the type for a server with an address of your own, enter that address, and leave the key empty if the server has none. An address on your own computer is always allowed. An address on your local network, such as `http://192.168.1.20:11434/v1`, is allowed only after you turn on **Allow a private network address** in the connection.

## Two windows

If you edit the same connection in two windows, the second save is refused with a message that another window changed it. Close the dialog, open it again and repeat the change.
