package main

// DO NOT CHANGE THIS CACHE SIZE VALUE
const CACHE_SIZE int = 3
	
var hMap = map[int]int{}
var lArray = []int{0,0,0}

func Set(key int, value int) {
	
	_, exists := hMap[key]
	
	if !exists{
			
		if len(hMap) == CACHE_SIZE{
			repEle := lArray[CACHE_SIZE-1]
			delete(hMap, repEle)
		}
		
		hMap[key] = value	
		
		for i:=CACHE_SIZE-2; i>=0; i-- {
			lArray[i+1] = lArray[i]
		}

		lArray[0] = key
	}else{
		var repEle int
		hMap[key] = value	
		
		for i:=1; i<CACHE_SIZE;i++{
			if lArray[i] == key{
				repEle = i
				break
			}
		}
		for j:=repEle-1;j>=0;j--{
			lArray[j+1] = lArray[j]
		}
		
		lArray[0] = key
	}
}

func Get(key int) int {
	iVal, exists := hMap[key]
	
	if exists{
		return iVal
	}
	
	return -1
}

