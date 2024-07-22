# fnf-practice
It's a program to practice Friday Night Funkin songs.
You can add songs from other [Friday Night Funkin](https://github.com/FunkinCrew/Funkin) mods and practice them.

# Features
### Demo Video ([FUN](https://youtu.be/2TPd4YU-eEs?t=451) by [bb-panzu](https://www.youtube.com/@bbpanzu213))

https://github.com/user-attachments/assets/035c5b0f-308c-4b83-90f7-adcf64919f8e

- Review your mistakes.
- Change the audio playback speed.
- Use rewind feature to automatically rewind on mistake.

# How to add songs

https://github.com/user-attachments/assets/4ac94f4a-9cbc-4c5f-a81a-3b3e9e6600c0

When you first launch the app, there will be no songs available.

Press enter to select 'Search Directory' menu. File explorer will show up.

Navigate to the directory where your other Friday Night Funking mod is located. Select it to load songs in that directory.

# Tested mods
- [Psych Engine](https://gamebanana.com/mods/309789)
- [Miku Mod](https://gamebanana.com/mods/44307)
- [FUNKIN' IS MAGIC](https://gamebanana.com/mods/380384)

# Limitations
### It doesn't support 'weird' notes like notes that hurt players when hit.
And it probably never will because from my understanding, different mods have their own way of implementing custom notes and I don't want to implement them one by one.

(Probably, maybe I will if I really like that mod.)

### Note hit system is probably quite different.
It's not exactly a 1 to 1 replication of other mods.

# Building from source
### Windows
On Windows, you need [Go](https://go.dev/) compiler and [tdm-gcc](https://jmeubank.github.io/tdm-gcc/)(for cgo) to build from the source. Once installed, run
```console
> build.bat
```
to build the app. (At least, that's how I do it.)

### Linux
On Linux, you need [Go](https://go.dev/) compiler and some c compiler. Also on Linux, you need gtk3 development package for [sqweek/dialog](https://github.com/sqweek/dialog) package(It's used to display file explorer).

Once installed, run
```console
> ./build.sh
```
to build the app.

Hopefully it'll build, *hopefully...*

# Why did you write in Go instead of Haxe?
Cause I like Go.
