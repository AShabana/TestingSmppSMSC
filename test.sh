echo this all work on smsc=Mobily and port= 3005 just for testing

cmd=$1
echo command is : $cmd

case $cmd in 
list)
    curl -s 'localhost:10000/listports' | jq '.[] | select(.Port == 3005)'
    ;;
freeze)
    curl -s 'localhost:10000/freezesmsc?smsc=Mobily' 
    ;;
unfreeze)
    curl -s 'localhost:10000/unfreezesmsc?smsc=Mobily' 
    ;;
unbind)
    curl -s 'localhost:10000/unbindall?smsc=Mobily' 
    ;;
*)
    ;;
esac

