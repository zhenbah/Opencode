import {
    Circle,
    Grid,
    Layout,
    Line,
    Node,
    Rect,
    Txt,
    makeScene2D,
  } from '@motion-canvas/2d';
  import {
    all,
    createEffect,
    createSignal,
    easeInOutBounce,
    linear,
    map,
    tween,
    waitFor,
  } from '@motion-canvas/core';
  import {createRef} from '@motion-canvas/core';
      // Generate more realistic synthetic S&P 500 data
import {logMethods} from './utils';

// Function to generate more realistic synthetic S&P 500 data
function generateRealisticSandPData(numPoints: number, startDate: Date = new Date(2000, 0, 1)) {
    const data = [];
    let currentValue = 1000; // A more realistic starting value for an index, though still arbitrary for simulation

    // S&P 500 historically averages around 10% annual return.
    // We'll convert this to a daily average for more granular simulation.
    // There are approximately 252 trading days in a year.
    const averageDailyReturn = 0.10 / 252;

    // Volatility (standard deviation) is also crucial.
    // Historical daily volatility for S&P 500 is roughly 1-1.5%.
    const dailyVolatility = 0.012; // 1.2% daily volatility

    let currentDate = new Date(startDate);

    for (let i = 0; i < numPoints; i++) {
        // Calculate a random daily return based on average and volatility
        // Using a normal distribution approximation for more realistic fluctuations
        // For simplicity, we'll use a basic random number for now.
        // A more advanced simulation would use a Box-Muller transform for true normal distribution.
        const randomFactor = Math.random() * 2 - 1; // Random number between -1 and 1

        // The daily change is influenced by the average daily return and random volatility
        // compounded on the current value.
        const dailyChange = currentValue * (averageDailyReturn + (randomFactor * dailyVolatility));
        currentValue += dailyChange;

        // Ensure value doesn't go negative (though highly unlikely with these parameters)
        if (currentValue < 0) {
            currentValue = 0.1; // Set to a small positive value if it somehow drops below zero
        }

        // Increment date for each data point
        // We'll simulate trading days, so skip weekends.
        currentDate.setDate(currentDate.getDate() + 1);
        while (currentDate.getDay() === 0 || currentDate.getDay() === 6) { // 0 is Sunday, 6 is Saturday
            currentDate.setDate(currentDate.getDate() + 1);
        }

        data.push({ time: new Date(currentDate), value: currentValue });
    }

    return data;
}

export default makeScene2D(function* (view) {
    // Signals
    const num_points = 1000;
    const sandp_data = generateRealisticSandPData(num_points);
    const TIME = 3.5;
    // Find min and max values for normalization
    const minValue = Math.min(...sandp_data.map(d => d.value));
    const maxValue = Math.max(...sandp_data.map(d => d.value));
    const minDateTimestamp = Math.min(...sandp_data.map(d => d.time.getTime())); // Get timestamps for min
    const maxDateTimestamp = Math.max(...sandp_data.map(d => d.time.getTime())); // Get timestamps for max
    
    const minDate = new Date(minDateTimestamp); // Convert timestamp back to Date object
    const maxDate = new Date(maxDateTimestamp); // Convert timestamp back to Date object
    const animationTime = createSignal(0);
    function animationTimetoDate(time: number, minDate: Date, maxDate: Date, TIME: number): Date {
        // 1. Get the timestamps for minDate and maxDate
        const minTimestamp = minDate.getTime(); // Milliseconds since epoch
        const maxTimestamp = maxDate.getTime();
    
        // 2. Calculate the ratio of the current animation time to the total animation duration
        // Ensure TIME is not zero to avoid division by zero errors
        if (TIME === 0) {
            // Handle this case: perhaps return minDate or throw an error
            console.warn("Total animation duration (TIME) is zero. Returning minDate.");
            return minDate;
        }
        const timeRatio = animationTime() / TIME;
    
        // 3. Calculate the target timestamp based on the ratio
        // The range of dates in milliseconds
        const dateRangeMillis = maxTimestamp - minTimestamp;
        // Add the proportional duration to the minTimestamp
        const targetTimestamp = minTimestamp + (dateRangeMillis * timeRatio);
    
        // 4. Convert the target timestamp back to a Date object
        const resultDate = new Date(targetTimestamp);
    
        return resultDate;
    }

    const currentGraphDate = createSignal(minDate);
    const effect = createEffect(
        () => {
            currentGraphDate(animationTimetoDate(animationTime(), minDate, maxDate, TIME))
        }
    )

    const value = createSignal(0);
    const dataIndex = createSignal(0);

 
    
    // Calculate grid dimensions
    const gridSize = 700;
    const gridHalfSize = gridSize / 2;
    const spacing = 100;
    const subgridSize = (spacing:number) =>gridSize- 1*spacing;
    // Grid is positioned at y=-30, so its bottom edge is at y=-30+gridHalfSize

    
    const rectref = createRef<Rect>();
    const line_vertical = createRef<Line>();
    const line_horizontal = createRef<Line>();
    const noderef = createRef<Node>();
    const horizontalNodeRef = createRef<Node>();
    // Animation date

    const gridref = createRef<Grid>();

    view.add(
      <Node y={-30} ref={noderef}>
        {/* Grid and animated point */}
        <Grid ref={gridref} size={gridSize} stroke={'#444'} lineWidth={3} spacing={spacing} start={0} end={0} >
          <Rect
            ref={rectref}
            layout
            size={100}
            offset={[0, 0]} // Center anchor point
            x={() => animationTime() / TIME* 500 - 300}
            y={() => {
              // Map the current S&P value to our y-coordinate space
              const idx = Math.floor(dataIndex() * (sandp_data.length - 1));
              const dataValue = sandp_data[idx].value;
              const dataValue_normalized = (dataValue - minValue) / (maxValue - minValue);
                
              // Start at the bottom of the grid and move up based on the normalized value
              // gridHalfSize is the distance from center to edge
              // Multiply by a factor less than 1 to keep within grid bounds
            //   return dataValue_normalized * -subgridSize(spacing);
            // return subgridSize(spacing);
            return dataValue_normalized * -subgridSize(spacing)/2  + subgridSize(spacing)/2  ;
            }}
            lineWidth={4} 
          >
            <Circle size={60} fill={'#C22929'} margin={20}></Circle>
          </Rect>
        </Grid>
        {/* Vertical */}
        <Node position={[-400, -400]}>
          {/* Axis */}
          <Line
            ref={line_vertical}
            lineWidth={4}
            points={[
              [0, 750],
              [0, 35],
            ]}
            stroke={'#DDD'}
            lineCap={'round'}
            endArrow
            arrowSize={15}
            start={0}
            end={0}
          ></Line>
  
          <Layout y={() => {
            // Use the same data mapping for the tracker
            const world_position = rectref().absolutePosition();
            const matrix = noderef().worldToLocal();
            console.log(`world position is ${world_position}`);
            const localPosition = world_position.transformAsPoint(matrix);
            console.log(`local position is ${localPosition}`);
            return localPosition.y + 400; // Add 400 to compensate for the inner Node's y-offset
          }}>
            <Txt
              fill={'#DDD'}
              text={() => {
                const idx = Math.floor(dataIndex() * (sandp_data.length - 1));
                return sandp_data[idx].value.toFixed(2);
              }}
              fontWeight={300}
              fontSize={30}
              x={-55}
              y={3}
            ></Txt>
            <Circle size={30} fill={'#DDD'}></Circle>
          </Layout>
          {/* Label */}
          <Txt
            y={400}
            x={-160}
            fontWeight={400}
            fontSize={40}
            padding={20}
            fontFamily={'Candara'}
            fill={'#DDD'}
            text={'S&P VALUE'}
          ></Txt>
        </Node>
  
        {/* Horizontal */}
        <Node position={[-500, -400]} ref={horizontalNodeRef}>
          {/* Axis */}
          <Line
            ref={line_horizontal}
            lineWidth={4}
            points={[
              [50, 800],
              [765, 800],
            ]}
            stroke={'#DDD'}
            lineCap={'round'}
            endArrow
            arrowSize={15}
            start={0}
            end={0}
          ></Line>
  
          {/* Tracker */}
          <Layout y={800} x={() => {
            // Get rectangle's world position
            const world_position = rectref().absolutePosition();
            // Transform to main node's local space
            const matrix = horizontalNodeRef().worldToLocal();
            const localPosition = world_position.transformAsPoint(matrix);
            // Adjust for the inner Node's offset (-400 on x-axis)
            return localPosition.x; // Add 400 to compensate for the inner Node's x-offset
          }}>
            <Circle size={30} fill={'#DDD'}></Circle>
            <Txt
              fill={'#DDD'}
              text={() => {
                const idx = Math.floor(dataIndex() * (sandp_data.length - 1));
                return sandp_data[idx].time.toString();
              }}
              fontWeight={300}
              fontSize={30}
              y={50}
            ></Txt>
          </Layout>
  
          {/* Label */}
          <Txt
            y={900}
            x={400}
            fontWeight={400}
            fontSize={50}
            padding={20}
            fontFamily={'Candara'}
            fill={'#DDD'}
            text={'date'}
          ></Txt>
        </Node>
      </Node>,
    );

    yield* gridref().end(1,2);
    yield* line_vertical().end(1,2);
    yield* line_horizontal().end(1,2);
    yield* waitFor(0.5);
    
    // Log information without using protected methods
    
    // Animate through the S&P data
    yield* all(
        animationTime(1, TIME, linear),
      dataIndex(1, TIME, easeInOutBounce)
    );
    
    yield* waitFor(0.8);
  });
  