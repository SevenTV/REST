# Personal Emotes | 7TV Developer Documentation

"Personal Emotes" is a system which allows an [Emote Set](https://docs.7tv.io/structures/emote-set) to be bound to a specific user, hereby granting access to using emotes not specific to a Channel.

In essence, it is similar to Twitch's Special, Affiliate & Partner emotes. They're emotes usable by specific users, regardless of the channel emotes active in the channel.

This page exists to guide developers into implementing this system into their application. This currently refers to an implementation on twitch.tv, it has not yet been decided how this may function on other platforms such as youtube.com.

## Glossary

**Reading Client / Reader**: The part of a client implementation that handles verifying incoming tokenized messages and rendering emotes in chat\
**Writing Client / Writer**: The part of a client implementation that allows the usage of personal emotes for the end user\
**Signing Authority**: The API which acts as the authority for token signatures\
**Public Key**: The public part of the asymmetrical token keypair that allows clients to verify incoming messages

The implementation of Personal Emotes can be split into two parts; reading and writing.

Reading is to show personal emotes in your application\
Writing is to allow end users to post their personal emotes via your application.

It is _not required_ for developers to implement writing. This only means users would not be able to send their personal emotes through your application.

## Reading Implementation

### Pre-requirements

In order to implement reading, you must make sure your program can:

- Access the `@client-nonce` tag contained in Twitch IRC messages
- Verify and Decode [JSON Web Tokens](https://jwt.io/) (this may necessitate the use of a JWT library for your language)
- Ideally: have the ability to use another thread (in the case of a browser environment, a WebWorker)

### Getting the Public Key

The first thing clients should do (for example, when the app is launched) is to **request the public key** from the server.\
Because the keypair is relatively weak, it may be regenerated occasionally. As such, you must use a REST API endpoint to request it. (for example, when your app launches)

#### Get Developers Public Key

> GET `7tv.io/v3/developers/public-key`
```json
{
    "publickey": "LS0tLS1CRUdJTiBQVUJMSUMgS0VZLS0tLS0KICAgICAgICBNRmt3RXdZSEtvWkl6ajBDQVFZSUtvWkl6ajBEQVFjRFFnQUVhdjJKMDRLUy83N3hhSVcreFMwZEdGRXA4V21ZCiAgICAgICAgRHpRd09nUHVQL1FmLyt6SzBHb0hEOGJhMGRlNmRjOCtDS2hBNlY3MGtudTlBNkI2aVBoTWVQbHpHdz09CiAgICAgICAgLS0tLS1FTkQgUFVCTElDIEtFWS0tLS0tCg==%"
}
```
The endpoint will send the public key as base64-encoded. Make sure to decode this back into clear text before passing it to your JWT library/implementation.

### Parsing Incoming Messages

With the ECDSA Public Key now acquired, the next thing to do is begin parsing the `@client-nonce` tag in the header of messages within Twitch's IRC.\
Within this tag, messages which contain 7TV-specific data will feature a string similar to this:
```
@client-nonce=123456789,&7TV{<token>},someotherdata
```
The data contained between `&7TV{...}` is an asymmetric [JSON Web Token](https://jwt.io/).

The token can be parsed out with a Regular Expression, such as:
```regexp
(?<=\&7TV\{)([A-Za-z0-9-_]*\.[A-Za-z0-9-_]*\.[A-Za-z0-9-_]*)(?=\})
```
(_Note that a Negative Lookbehind is used in this example, which is only [70% supported](https://caniuse.com/js-regexp-lookbehind) in web browsers._)

With this token, as well as the public key, it's now possible to **verify**!

### Verifying Message Tokens

This is the point where you'll need a proper JWT library (or if you're that kind of person, [your own implementation](https://datatracker.ietf.org/doc/html/rfc7519)).

You should now be able to use your library's verify function, pass the token alongside the public key.\
If an error is thrown, simply skip and display the message normally.

Once verified, you should now have a payload, with data ressembling the following:
```json
{
    "t": 11, // the token type, for twitch IRC tokens this will always be 11
    "u": "24377667" // a twitch user ID, corresponding to the user this token was signed for
    "es": [1, 24, 777] // a list of Emote Set IDs (es = emote sets)
}
```
