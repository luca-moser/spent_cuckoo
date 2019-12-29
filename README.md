# Spent Cuckoo

A repo which evaluates how cuckoo filters can be used to store spent addresses.

At the current spent addresses count of ~13 million spent addresses, a cuckoo filter with 8-bit fingerprints,
4 entries per bucket and a capacity of 50 million (consumes 68 megabytes on disk), has a false-positive-probability of 0.6%. At a capacity of 100
million (131 megabytes on disk), the FPP drops to 0.3% with 13 million inserted spent addresses.

```
loading spent addresses...
Loading snapshot file...
populating the cuckoo filter with 12927520 spent addresses...
took 15.1358402s
serializing cuckoo filter to file...
took 204.0675ms
now looking up every spent address in the cuckoo filter
12927519/12927520
took 3m54.2983371s
now looking up 10000000 randomly generated spent addresses
9999999/10000000/(falsePositive=60035)/(skippedDup=0)
took 3m32.3393296s
```

```
loading spent addresses...
Loading snapshot file...
populating the cuckoo filter with 12927520 spent addresses...
populated the cuckoo filter with 12927520 items
took 14.5449987s
serializing cuckoo filter to file...
took 503.6244ms
now looking up 10000000 randomly generated spent addresses
9999999/10000000/(falsePositive=30220)/(skippedDup=0)
took 3m37.0700739s
```