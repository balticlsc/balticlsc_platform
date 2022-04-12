kubectl -n icekube get pods | tail -1 | awk '{print $1;}' | xargs kubectl -n icekube delete pod
