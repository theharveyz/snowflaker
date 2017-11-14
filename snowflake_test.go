package snowflake

import "testing"
import "fmt"
import "os"
import "sync"

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}

func TestGetTime(t *testing.T) {
	fmt.Println(getUnixMillisecond())
}

func TestNextID(t *testing.T) {
	sf, _ := New(1)
	for i := 0; i < 10; i++ {
		fmt.Println(sf.NextID())
	}
}

func TestMutliThreadNextID(t *testing.T) {
	sf, _ := New(2)
	var wg sync.WaitGroup
	ids := make([]int64, 0, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id, _ := sf.NextID()
			ids = append(ids, id)
			fmt.Println("G:", i)
		}(i)
	}
	wg.Wait()
	fmt.Println(ids)
}
