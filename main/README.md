# The implementation package

- [The main function file](state_transition/state_processor.go)
- [The main function data types](state_transition/types.go)

## Rationale
TBD


```
Given:
var1 == var2 == var3 == var4 == 0

Transactions:
0:
    var1 = 1        // RACE_CONDITION_1

1:
    var1 = var1 + 1 // RACE_CONDITION_1
                    // this is why we must repeatedly run the sequential set to discover more conflicts
    ... // A LOT of computations here + likely ABORT (according to the Taraxa algorithm)
    if var1 == 2
        var2 = 2

2:
    if var2 == 2
        var3 = 1
    
3:
    if var3 == 1
        var4 = 1
    else
        var4 = 2

Ethereum result:
concurrent_set == {}
sequential_set == {0, 1, 2, 3}
var1 == 2
var2 == 2
var3 == 1
var4 == 1

Taraxa result:
concurrent_set == {2, 3}
sequential_set == {0, 1}
var1 == 2
var2 == 2
var3 == 0
var4 == 2
```