#!/bin/sh -e

die() {
    echo $@ >&2
    exit 1
}

if ! type java > /dev/null; then
    die "need java"
fi

PLANTUML_JAR=${PLANTUML_JAR:-"plantuml.jar"}

if [ ! -f $PLANTUML_JAR ]; then
    echo "plantuml is not found, downloading"
    curl -Lo $PLANTUML_JAR http://sourceforge.net/projects/plantuml/files/plantuml.jar/download
fi

exec java -jar $PLANTUML_JAR doc/*.puml
