package main

func search(grid [][]int, x int, y int){
    
    if x<0 || y<0 || x>len(grid)-1 || y>len(grid[0])-1 {
        return
    }
 
    if grid[x][y] != 1 { 
    	return
	} 
    
    grid[x][y] = 0
 
    search(grid, x-1, y)
    search(grid, x+1, y)
    search(grid, x, y-1)
    search(grid, x, y+1)
}

func CountIslands(grid [][]int) int {
    if len(grid)==0 || len(grid[0])==0 {
		return 0
	}
    count := 0
 
    for x, _ := range grid {
        for y, _ := range grid[x] {
            if grid[x][y] == 1 {
                count = count + 1
                search(grid, x, y)
            }
        }
    }
    return count
}
 