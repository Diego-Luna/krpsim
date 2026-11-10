  #!/bin/sh
  set -e
  mode=$1
  mkdir -p .golden
  for f in resources/* configs/*.krp src/system/test-config/*.krp; do
      [ -f "$f" ] || continue
      out=.golden/$(basename "$f").out
      if [ "$mode" = record ]; then
          ./krpsim "$f" 1 > "$out"
      else
          ./krpsim "$f" 1 | diff -q - "$out" > /dev/null || { echo "changed: $f"; exit 1; }
      fi
  done
  echo ok